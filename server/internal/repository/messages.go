package repository

import (
	"context"
	"database/sql"
	"time"

	"campus-fellow/server/internal/model"
)

func (r *Repository) Conversation(ctx context.Context, id, user int64) (model.Conversation, error) {
	var c model.Conversation
	err := r.db.QueryRowContext(ctx, "SELECT id,user_a_id,user_b_id FROM conversations WHERE id=? AND (user_a_id=? OR user_b_id=?)", id, user, user).Scan(&c.ID, &c.UserAID, &c.UserBID)
	return c, err
}

func (r *Repository) Conversations(ctx context.Context, user int64) ([]model.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,u.id,u.username,u.nickname,u.avatar,u.status,
 COALESCE((SELECT content FROM messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),'申请通过，发句问候吧'),
 COALESCE((SELECT created_at FROM messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),c.created_at) AS last_message_at,
 (SELECT COUNT(*) FROM messages m WHERE m.conversation_id=c.id AND m.sender_id<>? AND m.read_at IS NULL),p.title
 FROM conversations c JOIN users u ON u.id=CASE WHEN c.user_a_id=? THEN c.user_b_id ELSE c.user_a_id END
 JOIN applications a ON a.id=c.application_id JOIN mate_posts p ON p.id=a.mate_post_id
 WHERE c.user_a_id=? OR c.user_b_id=? ORDER BY last_message_at DESC,c.id DESC`, user, user, user, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Conversation{}
	for rows.Next() {
		var c model.Conversation
		var avatar sql.NullString
		var last time.Time
		if err := rows.Scan(&c.ID, &c.OtherUser.ID, &c.OtherUser.Username, &c.OtherUser.Nickname, &avatar, &c.OtherUser.Status, &c.LastMessage, &last, &c.UnreadCount, &c.PostTitle); err != nil {
			return nil, err
		}
		c.OtherUser.Avatar = optionalString(avatar)
		c.LastMessageAt = timestamp(last)
		items = append(items, c)
	}
	return items, rows.Err()
}

const messageSelect = `SELECT m.id,m.content,m.created_at,m.read_at,u.id,u.nickname,u.avatar FROM messages m JOIN users u ON u.id=m.sender_id `

func scanMessage(row scanner) (model.Message, error) {
	var m model.Message
	var created time.Time
	var read sql.NullTime
	var avatar sql.NullString
	err := row.Scan(&m.ID, &m.Content, &created, &read, &m.Sender.ID, &m.Sender.Nickname, &avatar)
	m.CreatedAt = timestamp(created)
	m.ReadAt = optionalTime(read)
	m.Sender.Avatar = optionalString(avatar)
	return m, err
}
func (r *Repository) Messages(ctx context.Context, id, before int64, limit int) ([]model.Message, error) {
	query := messageSelect + " WHERE m.conversation_id=?"
	args := []any{id}
	if before > 0 {
		query += " AND m.id<?"
		args = append(args, before)
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query+" ORDER BY m.id DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, rows.Err()
}
func (r *Repository) CreateMessage(ctx context.Context, conversation, user int64, content string) (model.Message, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO messages(conversation_id,sender_id,content) VALUES (?,?,?)", conversation, user, content)
	if err != nil {
		return model.Message{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return model.Message{}, err
	}
	return scanMessage(r.db.QueryRowContext(ctx, messageSelect+" WHERE m.id=?", id))
}
func (r *Repository) ReadMessages(ctx context.Context, conversation, user int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE messages SET read_at=CURRENT_TIMESTAMP(3) WHERE conversation_id=? AND sender_id<>? AND read_at IS NULL", conversation, user)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
