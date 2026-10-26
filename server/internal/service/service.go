package service

import (
	"database/sql"
	"errors"
	"net/http"

	"campus-fellow/server/internal/repository"
)

type Error struct {
	Status  int
	Message string
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
	Repo   *repository.Repository
	Tokens *Tokens
	Events Publisher
}

func New(repo *repository.Repository, tokens *Tokens, events Publisher) *Service {
	return &Service{Repo: repo, Tokens: tokens, Events: events}
}
