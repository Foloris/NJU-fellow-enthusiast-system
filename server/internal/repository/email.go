package repository

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"math"
	"time"

	"campus-fellow/server/internal/model"
)

var (
	ErrEmailUsed    = errors.New("email already bound")
	ErrCodeInvalid  = errors.New("verification code invalid or expired")
	ErrCodeAttempts = errors.New("verification attempts exhausted")
)

type SendLimitError struct{ Seconds int }

func (e *SendLimitError) Error() string { return "verification sending rate exceeded" }

// Locks source then email in a consistent order; limits work across restarts and instances.
func (r *Repository) ReserveEmailCode(ctx context.Context, email, hash, nonce, sourceHash string) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		var now time.Time
		if err := tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(3)").Scan(&now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO email_verification_send_limits(source_hash,window_start,sent_count) VALUES (?,?,0)", sourceHash, now); err != nil {
			return err
		}
		var start time.Time
		var sent int
		if err := tx.QueryRowContext(ctx, "SELECT window_start,sent_count FROM email_verification_send_limits WHERE source_hash=? FOR UPDATE", sourceHash).Scan(&start, &sent); err != nil {
			return err
		}
		if now.Sub(start) >= time.Hour {
			start = now
			sent = 0
		}
		if sent >= 60 {
			return &SendLimitError{Seconds: max(1, int(math.Ceil(start.Add(time.Hour).Sub(now).Seconds())))}
		}
		var used int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_emails WHERE email=?", email).Scan(&used); err != nil {
			return err
		}
		if used != 0 {
			return ErrEmailUsed
		}
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO email_verifications(email,code_hash,nonce,expires_at,next_send_at) VALUES (?,?,?,?,?)`, email, hash, nonce, now, now); err != nil {
			return err
		}
		var next time.Time
		if err := tx.QueryRowContext(ctx, "SELECT next_send_at FROM email_verifications WHERE email=? FOR UPDATE", email).Scan(&next); err != nil {
			return err
		}
		if next.After(now) {
			return &SendLimitError{Seconds: max(1, int(math.Ceil(next.Sub(now).Seconds())))}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE email_verifications SET code_hash=?,nonce=?,state='sending',expires_at=?,next_send_at=?,attempts=0 WHERE email=?", hash, nonce, now, now.Add(time.Minute), email); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE email_verification_send_limits SET window_start=?,sent_count=? WHERE source_hash=?", start, sent+1, sourceHash)
		return err
	})
}

func (r *Repository) ActivateEmailCode(ctx context.Context, email, nonce string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE email_verifications SET state='ready',expires_at=DATE_ADD(UTC_TIMESTAMP(3),INTERVAL 60 SECOND),next_send_at=DATE_ADD(UTC_TIMESTAMP(3),INTERVAL 60 SECOND) WHERE email=? AND nonce=? AND state='sending'`, email, nonce)
	return affected(result, err)
}
func (r *Repository) CancelEmailCode(ctx context.Context, email, nonce string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM email_verifications WHERE email=? AND nonce=? AND state='sending'", email, nonce)
	return err
}

// Creating a user, binding the mailbox and consuming the code commit together.
// A wrong code commits its failed-attempt counter without creating an account.
func (r *Repository) RegisterVerified(ctx context.Context, name, passwordHash, nickname, email, codeHash string) (model.User, error) {
	var id int64
	var verificationError error
	err := r.transaction(ctx, func(tx *sql.Tx) error {
		var savedHash, state string
		var attempts int
		var valid bool
		err := tx.QueryRowContext(ctx, "SELECT code_hash,state,attempts,expires_at>UTC_TIMESTAMP(3) FROM email_verifications WHERE email=? FOR UPDATE", email).Scan(&savedHash, &state, &attempts, &valid)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCodeInvalid
		}
		if err != nil {
			return err
		}
		if state != "ready" || !valid {
			return ErrCodeInvalid
		}
		if attempts >= 5 {
			return ErrCodeAttempts
		}
		if subtle.ConstantTimeCompare([]byte(savedHash), []byte(codeHash)) != 1 {
			if _, err := tx.ExecContext(ctx, "UPDATE email_verifications SET attempts=attempts+1 WHERE email=?", email); err != nil {
				return err
			}
			verificationError = ErrCodeInvalid
			if attempts+1 >= 5 {
				verificationError = ErrCodeAttempts
			}
			return nil
		}
		result, err := tx.ExecContext(ctx, "INSERT INTO users(username,password_hash,nickname) VALUES (?,?,?)", name, passwordHash, nickname)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO user_emails(user_id,email) VALUES (?,?)", id, email); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE email_verifications SET state='consumed' WHERE email=?", email)
		return err
	})
	if err != nil {
		return model.User{}, err
	}
	if verificationError != nil {
		return model.User{}, verificationError
	}
	return r.UserByID(ctx, id)
}
