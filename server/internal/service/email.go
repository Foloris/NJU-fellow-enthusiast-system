package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"campus-fellow/server/internal/repository"
)

type RegistrationSettings struct {
	EmailDomain            string `json:"emailDomain"`
	CodeLifetimeSeconds    int    `json:"codeLifetimeSeconds"`
	ResendAfterSeconds     int    `json:"resendAfterSeconds"`
	EmailDeliveryAvailable bool   `json:"emailDeliveryAvailable"`
}

func (s *Service) RegistrationSettings() RegistrationSettings {
	return RegistrationSettings{EmailDomain: s.emailDomain, CodeLifetimeSeconds: 60, ResendAfterSeconds: 60, EmailDeliveryAvailable: s.mailer != nil}
}

type EmailCodeInput struct {
	Email string `json:"email"`
}
type EmailCodeResult struct {
	Message    string `json:"message"`
	ExpiresIn  int    `json:"expiresIn"`
	RetryAfter int    `json:"retryAfter"`
}

var verificationCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

func (s *Service) campusEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || strings.ContainsAny(email, "\r\n\t ") {
		return "", fail(400, "请输入有效的校园邮箱地址")
	}
	for _, r := range email {
		if r > 127 {
			return "", fail(400, "邮箱地址仅支持英文字符")
		}
	}
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain != s.emailDomain {
		return "", fail(400, "目前仅支持 @"+s.emailDomain+" 校园邮箱")
	}
	return email, nil
}
func (s *Service) codeHash(email, code string) string {
	hash := hmac.New(sha256.New, s.Tokens.secret)
	hash.Write([]byte("registration-email\x00" + email + "\x00" + code))
	return hex.EncodeToString(hash.Sum(nil))
}

func (s *Service) SendEmailCode(ctx context.Context, raw, source string) (EmailCodeResult, error) {
	email, err := s.campusEmail(raw)
	if err != nil {
		return EmailCodeResult{}, err
	}
	if s.mailer == nil {
		return EmailCodeResult{}, fail(503, "邮件发送服务尚未配置，请联系管理员")
	}
	code, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return EmailCodeResult{}, err
	}
	value := fmt.Sprintf("%06d", code.Int64())
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return EmailCodeResult{}, err
	}
	requestID := hex.EncodeToString(nonce)
	sourceDigest := sha256.Sum256([]byte(source))
	err = s.Repo.ReserveEmailCode(ctx, email, s.codeHash(email, value), requestID, hex.EncodeToString(sourceDigest[:]))
	if errors.Is(err, repository.ErrEmailUsed) {
		return EmailCodeResult{}, fail(409, "这个邮箱已经绑定过账号")
	}
	var limit *repository.SendLimitError
	if errors.As(err, &limit) {
		return EmailCodeResult{}, &Error{Status: 429, Message: fmt.Sprintf("发送过于频繁，请在%d秒后重试", limit.Seconds), RetryAfter: limit.Seconds}
	}
	if err != nil {
		return EmailCodeResult{}, err
	}
	if err = s.mailer.SendVerification(ctx, email, value); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if cleanupErr := s.Repo.CancelEmailCode(cleanup, email, requestID); cleanupErr != nil {
			slog.Error("cancel email verification", "error", cleanupErr)
		}
		slog.Error("SMTP verification delivery failed", "error", err)
		return EmailCodeResult{}, fail(503, "验证码邮件发送失败，请稍后重试或联系管理员")
	}
	if err = s.Repo.ActivateEmailCode(ctx, email, requestID); err != nil {
		return EmailCodeResult{}, err
	}
	return EmailCodeResult{Message: "验证码已发送，60秒内有效，请检查收件箱和垃圾邮件", ExpiresIn: 60, RetryAfter: 60}, nil
}
