package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"campus-fellow/server/internal/model"
)

const exchangeSelect = `SELECT p.id,p.author_id,p.kind,p.title,p.content,p.image_url,p.status,p.created_at,p.updated_at,
 (SELECT COUNT(*) FROM likes l WHERE l.post_id=p.id),
 (SELECT COUNT(*) FROM favorites f WHERE f.post_id=p.id),
 (SELECT COUNT(*) FROM comments c WHERE c.post_id=p.id),
 EXISTS(SELECT 1 FROM likes l WHERE l.post_id=p.id AND l.user_id=?),
 EXISTS(SELECT 1 FROM favorites f WHERE f.post_id=p.id AND f.user_id=?),
 u.id,u.username,u.nickname,u.avatar FROM exchange_posts p JOIN users u ON u.id=p.author_id `

func scanExchange(row scanner) (model.Exchange, error) {
	var p model.Exchange
	var image, avatar sql.NullString
	var created, updated time.Time
	err := row.Scan(&p.ID, &p.AuthorID, &p.Kind, &p.Title, &p.Content, &image, &p.Status, &created, &updated, &p.LikeCount, &p.FavoriteCount, &p.CommentCount, &p.Liked, &p.Favorited, &p.Author.ID, &p.Author.Username, &p.Author.Nickname, &avatar)
	p.ImageURL = optionalString(image)
	p.Author.Avatar = optionalString(avatar)
	p.CreatedAt = timestamp(created)
	p.UpdatedAt = timestamp(updated)
	return p, err
}

func (r *Repository) Exchanges(ctx context.Context, viewer int64, f model.Filter) (model.Page[model.Exchange], error) {
	conditions := []string{}
	args := []any{}
	if f.Mine {
		conditions = append(conditions, "p.author_id=?")
		args = append(args, viewer)
	} else {
		conditions = append(conditions, "p.status='published'", "u.status='active'")
	}
	if f.Query != "" {
		conditions = append(conditions, "(p.title LIKE ? OR p.content LIKE ?)")
		args = append(args, "%"+f.Query+"%", "%"+f.Query+"%")
	}
	if f.Kind == "share" || f.Kind == "news" {
		conditions = append(conditions, "p.kind=?")
		args = append(args, f.Kind)
	}
	where := "WHERE " + strings.Join(conditions, " AND ")
	page := model.Page[model.Exchange]{Items: []model.Exchange{}, Page: f.Page, PageSize: f.PageSize}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exchange_posts p JOIN users u ON u.id=p.author_id "+where, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	params := append([]any{viewer, viewer}, args...)
	params = append(params, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.QueryContext(ctx, exchangeSelect+where+" ORDER BY CASE p.kind WHEN 'news' THEN 0 ELSE 1 END,p.created_at DESC,p.id DESC LIMIT ? OFFSET ?", params...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanExchange(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, p)
	}
	return page, rows.Err()
}

func (r *Repository) Exchange(ctx context.Context, id, viewer int64) (model.Exchange, error) {
	return scanExchange(r.db.QueryRowContext(ctx, exchangeSelect+" WHERE p.id=?", viewer, viewer, id))
}
func (r *Repository) CreateExchange(ctx context.Context, author int64, in model.ExchangeInput) (int64, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO exchange_posts(author_id,kind,title,content,image_url) VALUES (?,?,?,?,?)", author, in.Kind, in.Title, in.Content, in.ImageURL)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (r *Repository) UpdateExchange(ctx context.Context, id int64, in model.ExchangeInput) error {
	_, err := r.db.ExecContext(ctx, "UPDATE exchange_posts SET kind=?,title=?,content=?,image_url=?,updated_at=CURRENT_TIMESTAMP(3) WHERE id=?", in.Kind, in.Title, in.Content, in.ImageURL, id)
	return err
}
func (r *Repository) DeleteExchange(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM exchange_posts WHERE id=?", id)
	return err
}

func (r *Repository) Comments(ctx context.Context, id int64) ([]model.Comment, error) {
	items := []model.Comment{}
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.content,c.created_at,u.id,u.username,u.nickname,u.avatar FROM comments c JOIN users u ON u.id=c.user_id WHERE c.post_id=? ORDER BY c.created_at,c.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c model.Comment
		var created time.Time
		var avatar sql.NullString
		if err := rows.Scan(&c.ID, &c.Content, &created, &c.Author.ID, &c.Author.Username, &c.Author.Nickname, &avatar); err != nil {
			return nil, err
		}
		c.CreatedAt = timestamp(created)
		c.Author.Avatar = optionalString(avatar)
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *Repository) CreateComment(ctx context.Context, post, user int64, content string) (int64, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO comments(post_id,user_id,content) VALUES (?,?,?)", post, user, content)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (r *Repository) CommentOwner(ctx context.Context, id int64) (int64, error) {
	var owner int64
	err := r.db.QueryRowContext(ctx, "SELECT user_id FROM comments WHERE id=?", id).Scan(&owner)
	return owner, err
}
func (r *Repository) DeleteComment(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM comments WHERE id=?", id)
	return err
}

func (r *Repository) ToggleReaction(ctx context.Context, post, user int64, favorite bool) (active bool, count int, err error) {
	table := "likes"
	if favorite {
		table = "favorites"
	}
	err = r.transaction(ctx, func(tx *sql.Tx) error {
		// Serialize toggles on the post so concurrent clicks cannot race the unique key.
		var id int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM exchange_posts WHERE id=? AND status='published' FOR UPDATE", post).Scan(&id); err != nil {
			return err
		}
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE post_id=? AND user_id=?", post, user).Scan(&exists); err != nil {
			return err
		}
		active = exists == 0
		if active {
			if _, err := tx.ExecContext(ctx, "INSERT INTO "+table+"(post_id,user_id) VALUES (?,?)", post, user); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE post_id=? AND user_id=?", post, user); err != nil {
				return err
			}
		}
		return tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE post_id=?", post).Scan(&count)
	})
	return
}
