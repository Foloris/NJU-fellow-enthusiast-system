package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"campus-fellow/server/internal/config"
)

type SMTP struct {
	host, port, username, password, from, mode string
	tlsConfig                                  *tls.Config
}

// Attachment is an image attached to an outgoing message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func NewSMTP(c config.Config) *SMTP {
	return &SMTP{host: c.SMTPHost, port: c.SMTPPort, username: c.SMTPUsername, password: c.SMTPPassword, from: c.SMTPFrom, mode: c.SMTPTLSMode, tlsConfig: &tls.Config{ServerName: c.SMTPHost, MinVersion: tls.VersionTLS12}}
}

func (s *SMTP) SendVerification(ctx context.Context, to, code string) error {
	message, err := verificationMessage(s.from, to, code)
	if err != nil {
		return err
	}
	return s.deliver(ctx, to, message)
}

func (s *SMTP) SendFeedback(ctx context.Context, to, subject, body string, attachments []Attachment) error {
	message, err := feedbackMessage(s.from, to, subject, body, attachments)
	if err != nil {
		return err
	}
	return s.deliver(ctx, to, message)
}

func (s *SMTP) deliver(ctx context.Context, to string, message []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	connection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(s.host, s.port))
	if err != nil {
		return err
	}
	defer connection.Close()
	deadline, _ := ctx.Deadline()
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	rawConnection := connection
	stop := context.AfterFunc(ctx, func() { rawConnection.Close() })
	defer stop()
	if s.mode == "tls" {
		secured := tls.Client(connection, s.tlsConfig)
		if err := secured.HandshakeContext(ctx); err != nil {
			return err
		}
		connection = secured
	}
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if s.mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not offer required STARTTLS")
		}
		if err := client.StartTLS(s.tlsConfig); err != nil {
			return err
		}
	}
	if _, ok := client.TLSConnectionState(); !ok {
		return errors.New("SMTP connection must use TLS")
	}
	if err := client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
		return err
	}
	from, err := mail.ParseAddress(s.from)
	if err != nil {
		return err
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	// DATA's 250 reply confirms acceptance; a broken QUIT must not invalidate delivered mail.
	_ = client.Quit()
	return nil
}

func verificationMessage(from, to, code string) ([]byte, error) {
	var message bytes.Buffer
	if err := writeHeaders(&message, from, to, "搭个伴 · 校园邮箱验证码", "text/plain; charset=UTF-8", "quoted-printable"); err != nil {
		return nil, err
	}
	writer := quotedprintable.NewWriter(&message)
	if _, err := fmt.Fprintf(writer, "你的注册验证码是：%s\r\n\r\n验证码有效时间为60秒，仅用于本次校园邮箱注册。\r\n如果不是你本人操作，请忽略此邮件。\r\n", code); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return message.Bytes(), nil
}

func feedbackMessage(from, to, subject, body string, attachments []Attachment) ([]byte, error) {
	var message bytes.Buffer
	if len(attachments) == 0 {
		if err := writeHeaders(&message, from, to, subject, "text/plain; charset=UTF-8", "quoted-printable"); err != nil {
			return nil, err
		}
		writer := quotedprintable.NewWriter(&message)
		if _, err := writer.Write([]byte(body)); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		return message.Bytes(), nil
	}

	boundary, err := randomToken()
	if err != nil {
		return nil, err
	}
	if err := writeHeaders(&message, from, to, subject, `multipart/mixed; boundary="`+boundary+`"`, ""); err != nil {
		return nil, err
	}
	fmt.Fprintf(&message, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", boundary)
	writer := quotedprintable.NewWriter(&message)
	if _, err := writer.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	for _, attachment := range attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename})
		fmt.Fprintf(&message, "\r\n--%s\r\nContent-Type: %s\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: %s\r\n\r\n", boundary, contentType, disposition)
		writeBase64(&message, attachment.Data)
	}
	fmt.Fprintf(&message, "\r\n--%s--\r\n", boundary)
	return message.Bytes(), nil
}

func writeHeaders(message *bytes.Buffer, from, to, subject, contentType, transferEncoding string) error {
	nonce, err := randomToken()
	if err != nil {
		return err
	}
	fmt.Fprintf(message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@campus-fellow>\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n",
		from, to, mime.QEncoding.Encode("UTF-8", subject), time.Now().UTC().Format(time.RFC1123Z), nonce, contentType)
	if transferEncoding != "" {
		fmt.Fprintf(message, "Content-Transfer-Encoding: %s\r\n", transferEncoding)
	}
	message.WriteString("\r\n")
	return nil
}

func writeBase64(message *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		message.WriteString(encoded[:76])
		message.WriteString("\r\n")
		encoded = encoded[76:]
	}
	message.WriteString(encoded)
	message.WriteString("\r\n")
}

func randomToken() (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce[:]), nil
}
