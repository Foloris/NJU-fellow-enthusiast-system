package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"strings"
	"testing"
)

func TestSMTPFeedbackDelivery(t *testing.T) {
	certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := certificateServer.TLS.Certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(certificateServer.Certificate())
	certificateServer.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverTLS := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	listener = tls.NewListener(listener, serverTLS)
	done := make(chan smtpReceipt, 1)
	go func() { done <- receiveSMTP(listener, serverTLS, "tls") }()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	sender := &SMTP{host: host, port: port, username: "sender@test.example", password: "smtp-application-password", from: "sender@test.example", mode: "tls", tlsConfig: &tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12}}
	image := []byte("\x89PNG\r\n\x1a\nfake-screenshot-bytes")
	if err := sender.SendFeedback(context.Background(), "student@smail.nju.edu.cn", "[问题反馈] 搭个伴", "在发现页点击私信后页面空白\r\n请尽快修复。", []Attachment{{Filename: "截图1.png", ContentType: "image/png", Data: image}}); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(result.message))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil || subject != "[问题反馈] 搭个伴" {
		t.Fatalf("invalid subject: %q", subject)
	}
	if message.Header.Get("To") != "student@smail.nju.edu.cn" {
		t.Fatal("wrong recipient")
	}
	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" || params["boundary"] == "" {
		t.Fatalf("invalid content type: %q", message.Header.Get("Content-Type"))
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	text, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "在发现页点击私信后页面空白") || !strings.Contains(string(body), "请尽快修复。") {
		t.Fatalf("incorrect feedback body: %s", body)
	}
	attachment, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if attachment.Header.Get("Content-Type") != "image/png" || attachment.FileName() != "截图1.png" {
		t.Fatalf("invalid attachment headers: %v", attachment.Header)
	}
	decoded := new(bytes.Buffer)
	if _, err := decoded.ReadFrom(base64.NewDecoder(base64.StdEncoding, attachment)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Bytes(), image) {
		t.Fatal("attachment bytes changed in transit")
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatal("unexpected extra message part")
	}
}
