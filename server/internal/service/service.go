package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"campus-fellow/server/internal/repository"
)

type Error struct {
	Status     int
	Message    string
	RetryAfter int
}

func (e *Error) Error() string              { return e.Message }
func fail(status int, message string) error { return &Error{Status: status, Message: message} }
func missing(err error, message string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, message)
	}
	return err
}

type Service struct {
	Repo        *repository.Repository
	Tokens      *Tokens
	Events      Publisher
	emailDomain string
	mailer      VerificationMailer
}

type VerificationMailer interface {
	SendVerification(context.Context, string, string) error
}
type Option func(*Service)

func WithEmailVerification(domain string, mailer VerificationMailer) Option {
	return func(s *Service) { s.emailDomain = domain; s.mailer = mailer }
}

func New(repo *repository.Repository, tokens *Tokens, events Publisher, options ...Option) *Service {
	s := &Service{Repo: repo, Tokens: tokens, Events: events, emailDomain: "smail.nju.edu.cn"}
	for _, option := range options {
		option(s)
	}
	return s
}
