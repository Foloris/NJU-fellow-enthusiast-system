package service

import (
	"testing"
	"time"

	"campus-fellow/server/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

func TestTokensRejectInvalidSessions(t *testing.T) {
	tokens := NewTokens("test-secret-with-at-least-thirty-two-bytes")
	raw, err := tokens.Sign(model.User{ID: 42, Username: "alice", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if id, err := tokens.Verify(raw); err != nil || id != 42 {
		t.Fatalf("roundtrip: id=%d err=%v", id, err)
	}
	if _, err := NewTokens("different-secret-with-at-least-thirty-two-bytes").Verify(raw); err == nil {
		t.Fatal("accepted signature from another secret")
	}
	for _, tc := range []struct {
		name   string
		method jwt.SigningMethod
		claims claims
	}{
		{"expired", jwt.SigningMethodHS256, claims{ID: 42, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}},
		{"missing-expiry", jwt.SigningMethodHS256, claims{ID: 42}},
		{"wrong-algorithm", jwt.SigningMethodHS384, claims{ID: 42, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
		{"invalid-id", jwt.SigningMethodHS256, claims{ID: 0, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString(tokens.secret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tokens.Verify(raw); err == nil {
				t.Fatal("accepted invalid token")
			}
		})
	}
	if _, err := tokens.Verify("malformed"); err == nil {
		t.Fatal("accepted malformed token")
	}
}

func TestValidationMatchesBrowserLengths(t *testing.T) {
	if length("😀", 1, 1) || !length("😀", 2, 2) {
		t.Fatal("emoji length must match browser UTF-16 length")
	}
	in := model.ExchangeInput{Title: "  四字标题  ", Content: "  这是足够十个字符的正文内容。  "}
	if err := validateExchange(&in); err != nil {
		t.Fatal(err)
	}
	if in.Title != "四字标题" || in.Kind != "share" {
		t.Fatalf("normalization: %+v", in)
	}
	for _, tc := range []model.ExchangeInput{{Title: "短", Content: "正文不足"}, {Kind: "other", Title: "四字标题", Content: "这是足够十个字符的正文内容。"}} {
		if validateExchange(&tc) == nil {
			t.Fatal("accepted invalid exchange")
		}
	}
	mate := model.MateInput{Title: "周末活动邀请", Tags: []string{"  户外  "}, Details: "这是足够十个字符的搭子详情。", Contact: "测试联系", Duration: "周末"}
	if err := validateMate(&mate); err != nil {
		t.Fatal(err)
	}
	if mate.Tags[0] != "户外" {
		t.Fatal("tag not trimmed")
	}
	mate.Tags = nil
	if validateMate(&mate) == nil {
		t.Fatal("accepted empty tags")
	}
}

func TestContactsAreRestricted(t *testing.T) {
	contact := "私密联系方式"
	accepted := "accepted"
	pending := "pending"
	for _, tc := range []struct {
		name    string
		user    model.User
		status  *string
		visible bool
	}{
		{"guest", model.User{}, nil, false}, {"pending", model.User{ID: 2}, &pending, false}, {"accepted", model.User{ID: 2}, &accepted, true}, {"owner", model.User{ID: 1}, nil, true}, {"admin", model.User{ID: 3, Role: "admin"}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			post := model.Mate{AuthorID: 1, Contact: &contact, MyApplicationStatus: tc.status}
			protectContact(&post, tc.user)
			if (post.Contact != nil) != tc.visible {
				t.Fatal("contact visibility mismatch")
			}
			if !tc.visible && post.ContactHint == nil {
				t.Fatal("missing contact hint")
			}
		})
	}
}
