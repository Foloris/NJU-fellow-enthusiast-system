package mailer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

type smtpReceipt struct {
	message       []byte
	authenticated bool
	encrypted     bool
	err           error
}

func TestSMTPVerificationDelivery(t *testing.T) {
	certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := certificateServer.TLS.Certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(certificateServer.Certificate())
	certificateServer.Close()
	for _, mode := range []string{"tls", "starttls"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			serverTLS := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
			if mode == "tls" {
				listener = tls.NewListener(listener, serverTLS)
			}
			done := make(chan smtpReceipt, 1)
			go func() { done <- receiveSMTP(listener, serverTLS, mode) }()
			host, port, _ := net.SplitHostPort(listener.Addr().String())
			sender := &SMTP{host: host, port: port, username: "sender@test.example", password: "smtp-application-password", from: "sender@test.example", mode: mode, tlsConfig: &tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12}}
			if err := sender.SendVerification(context.Background(), "student@smail.nju.edu.cn", "012345"); err != nil {
				t.Fatal(err)
			}
			result := <-done
			if result.err != nil {
				t.Fatal(result.err)
			}
			if !result.authenticated || !result.encrypted {
				t.Fatal("SMTP was not authenticated and encrypted")
			}
			message, err := mail.ReadMessage(strings.NewReader(string(result.message)))
			if err != nil {
				t.Fatal(err)
			}
			subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
			if err != nil || !strings.Contains(subject, "验证码") {
				t.Fatal("invalid subject")
			}
			body, err := io.ReadAll(quotedprintable.NewReader(message.Body))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "012345") || !strings.Contains(string(body), "60秒") {
				t.Fatalf("incorrect verification message: %s", body)
			}
			if message.Header.Get("To") != "student@smail.nju.edu.cn" {
				t.Fatal("wrong recipient")
			}
		})
	}
}

func receiveSMTP(listener net.Listener, serverTLS *tls.Config, mode string) (result smtpReceipt) {
	connection, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer func() { connection.Close() }()
	connection.SetDeadline(time.Now().Add(5 * time.Second))
	protocol := textproto.NewConn(connection)
	if err := protocol.PrintfLine("220 test-smtp ready"); err != nil {
		result.err = err
		return
	}
	result.encrypted = mode == "tls"
	for {
		line, err := protocol.ReadLine()
		if err != nil {
			result.err = err
			return
		}
		switch {
		case strings.HasPrefix(line, "EHLO "):
			protocol.PrintfLine("250-test-smtp")
			if !result.encrypted {
				protocol.PrintfLine("250-STARTTLS")
			}
			protocol.PrintfLine("250 AUTH PLAIN")
		case line == "STARTTLS":
			protocol.PrintfLine("220 ready for TLS")
			secured := tls.Server(connection, serverTLS)
			if err := secured.Handshake(); err != nil {
				result.err = err
				return
			}
			connection = secured
			protocol = textproto.NewConn(connection)
			result.encrypted = true
		case strings.HasPrefix(line, "AUTH PLAIN "):
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "AUTH PLAIN "))
			if err != nil {
				result.err = err
				return
			}
			if string(decoded) != "\x00sender@test.example\x00smtp-application-password" || !result.encrypted {
				result.err = fmt.Errorf("incorrect SMTP authentication")
				return
			}
			result.authenticated = true
			protocol.PrintfLine("235 authenticated")
		case strings.HasPrefix(line, "MAIL FROM:"):
			if !result.authenticated || !strings.Contains(line, "sender@test.example") {
				result.err = fmt.Errorf("invalid envelope sender")
				return
			}
			protocol.PrintfLine("250 ok")
		case strings.HasPrefix(line, "RCPT TO:"):
			if !strings.Contains(line, "student@smail.nju.edu.cn") {
				result.err = fmt.Errorf("invalid envelope recipient")
				return
			}
			protocol.PrintfLine("250 ok")
		case line == "DATA":
			protocol.PrintfLine("354 send message")
			result.message, err = protocol.ReadDotBytes()
			if err != nil {
				result.err = err
				return
			}
			protocol.PrintfLine("250 accepted")
		case line == "QUIT":
			protocol.PrintfLine("221 goodbye")
			return
		default:
			result.err = fmt.Errorf("unexpected SMTP command")
			return
		}
	}
}

func TestSMTPRejectsMissingSTARTTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		connection.SetDeadline(time.Now().Add(3 * time.Second))
		protocol := textproto.NewConn(connection)
		protocol.PrintfLine("220 ready")
		protocol.ReadLine()
		protocol.PrintfLine("250 AUTH PLAIN")
		protocol.ReadLine()
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	sender := &SMTP{host: host, port: port, mode: "starttls", tlsConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	if err := sender.SendVerification(context.Background(), "student@smail.nju.edu.cn", "123456"); err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("unencrypted SMTP not rejected: %v", err)
	}
	<-done
}
