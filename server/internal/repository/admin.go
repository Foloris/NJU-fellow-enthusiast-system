package repository

import (
	"context"
	"database/sql"
	"time"

	"campus-fellow/server/internal/model"
)

func (r *Repository) Summary(ctx context.Context) (model.Summary, error) {
	var s model.Summary
	err := r.db.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM users WHERE role='user'),(SELECT COUNT(*) FROM users WHERE status='banned'),
 (SELECT COUNT(*) FROM exchange_posts WHERE status='published'),(SELECT COUNT(*) FROM mate_posts WHERE status='published'),
 (SELECT COUNT(*) FROM reports WHERE status='pending'),(SELECT COUNT(*) FROM conversations)`).Scan(&s.Users, &s.BannedUsers, &s.ExchangePosts, &s.MatePosts, &s.PendingReports, &s.ActiveConversations)
	return s, err
}

func (r *Repository) AdminUsers(ctx context.Context, q string) ([]model.AdminUser, error) {
	condition := "WHERE role='user'"
	args := []any{}
	if q != "" {
		condition = "WHERE username LIKE ? OR nickname LIKE ?"
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+userColumns+`,(SELECT COUNT(*) FROM exchange_posts p WHERE p.author_id=users.id)+(SELECT COUNT(*) FROM mate_posts p WHERE p.author_id=users.id) FROM users `+condition+" ORDER BY CASE status WHEN 'banned' THEN 0 ELSE 1 END,created_at DESC,id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.AdminUser{}
	for rows.Next() {
		var u model.AdminUser
		var avatar, email sql.NullString
		var created time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &avatar, &u.Bio, &u.Role, &u.Status, &created, &email, &u.PostCount); err != nil {
			return nil, err
		}
		u.Avatar = optionalString(avatar)
		u.Email = optionalString(email)
		u.CreatedAt = timestamp(created)
		items = append(items, u)
	}
	return items, rows.Err()
}

// Content table names come only from this allowlist, never from request text.
func contentTable(kind string) string {
	if kind == "mate" {
		return "mate_posts"
	}
	return "exchange_posts"
}
func (r *Repository) AdminContent(ctx context.Context, kind, q string) ([]model.AdminContent, error) {
	table, body, kindColumn := contentTable(kind), "content", "p.kind"
	if kind == "mate" {
		body = "details"
		kindColumn = "''"
	}
	where := ""
	args := []any{}
	if q != "" {
		where = " WHERE p.title LIKE ? OR p." + body + " LIKE ?"
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.title,p.`+body+`, `+kindColumn+`,p.status,p.created_at,u.nickname,u.username FROM `+table+` p JOIN users u ON u.id=p.author_id`+where+" ORDER BY p.created_at DESC,p.id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.AdminContent{}
	for rows.Next() {
		var p model.AdminContent
		var created time.Time
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Kind, &p.Status, &created, &p.AuthorNickname, &p.AuthorUsername); err != nil {
			return nil, err
		}
		p.CreatedAt = timestamp(created)
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *Repository) SetContentStatus(ctx context.Context, kind string, id int64, status string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE "+contentTable(kind)+" SET status=? WHERE id=?", status, id)
	return affected(result, err)
}
func (r *Repository) DeleteContent(ctx context.Context, kind string, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM "+contentTable(kind)+" WHERE id=?", id)
	return affected(result, err)
}
func affected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
