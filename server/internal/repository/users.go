package repository

import (
	"context"
	"database/sql"
	"time"

	"campus-fellow/server/internal/model"
)

const userColumns = "id, username, password_hash, nickname, avatar, bio, role, status, created_at, (SELECT email FROM user_emails WHERE user_id=users.id)"

func scanUser(row scanner) (model.User, error) {
	var u model.User
	var avatar, email sql.NullString
	var created time.Time
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &avatar, &u.Bio, &u.Role, &u.Status, &created, &email)
	u.Email = optionalString(email)
	u.Avatar = optionalString(avatar)
	u.CreatedAt = timestamp(created)
	return u, err
}

func (r *Repository) UserByID(ctx context.Context, id int64) (model.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id))
}
func (r *Repository) UserByUsername(ctx context.Context, name string) (model.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE username = ?", name))
}

func (r *Repository) UpdateProfile(ctx context.Context, id int64, nickname, bio string, avatar *string) (model.User, error) {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET nickname=?,bio=?,avatar=? WHERE id=?", nickname, bio, avatar, id)
	if err != nil {
		return model.User{}, err
	}
	return r.UserByID(ctx, id)
}

func (r *Repository) UserCounts(ctx context.Context, id int64) (exchanges, mates int, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM exchange_posts WHERE author_id=? AND status='published'), (SELECT COUNT(*) FROM mate_posts WHERE author_id=? AND status='published')`, id, id).Scan(&exchanges, &mates)
	return
}

func banUser(ctx context.Context, tx *sql.Tx, id int64, status string) error {
	if _, err := tx.ExecContext(ctx, "UPDATE users SET status=? WHERE id=?", status, id); err != nil {
		return err
	}
	if status == "banned" {
		for _, table := range []string{"exchange_posts", "mate_posts"} {
			if _, err := tx.ExecContext(ctx, "UPDATE "+table+" SET status='removed' WHERE author_id=?", id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Repository) SetUserStatus(ctx context.Context, id int64, status string) error {
	return r.transaction(ctx, func(tx *sql.Tx) error { return banUser(ctx, tx, id, status) })
}
