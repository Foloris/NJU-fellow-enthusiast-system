package service

import (
	"context"
	"testing"
)

func TestCampusEmailDomainValidation(t *testing.T) {
	s := New(nil, NewTokens("testing-email-secret-with-32-plus-bytes"), nil, WithEmailVerification("smail.nju.edu.cn", nil))
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{" student@SMAIL.NJU.EDU.CN ", true}, {"student+test@smail.nju.edu.cn", true},
		{"student@nju.edu.cn", false}, {"student@sub.smail.nju.edu.cn", false}, {"student@smail.nju.edu.cn.evil.example", false},
		{"Name <student@smail.nju.edu.cn>", false}, {"student@smail.nju.edu.cn\r\nBcc: bad@example.com", false}, {"学生@smail.nju.edu.cn", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			_, err := s.campusEmail(tc.input)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
	if s.codeHash("a@smail.nju.edu.cn", "123456") == s.codeHash("b@smail.nju.edu.cn", "123456") {
		t.Fatal("code digest is not bound to email")
	}
	if _, err := s.SendEmailCode(context.Background(), "student@smail.nju.edu.cn", "127.0.0.1"); err == nil {
		t.Fatal("missing SMTP accepted a send")
	}
	if settings := s.RegistrationSettings(); settings.EmailDeliveryAvailable || settings.CodeLifetimeSeconds != 60 {
		t.Fatal("incorrect public settings")
	}
}

func TestRegistrationRequiresEmailAndCode(t *testing.T) {
	s := New(nil, NewTokens("testing-email-secret-with-32-plus-bytes"), nil)
	for _, in := range []RegisterInput{
		{Username: "student", Password: "Testing123!", Nickname: "同学"},
		{Username: "student", Password: "Testing123!", Nickname: "同学", Email: "student@smail.nju.edu.cn"},
		{Username: "student", Password: "Testing123!", Nickname: "同学", Email: "student@smail.nju.edu.cn", EmailCode: "12345x"},
	} {
		if _, err := s.Register(context.Background(), in); err == nil {
			t.Fatal("unverified registration accepted")
		}
	}
}
