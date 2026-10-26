package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"campus-fellow/server/internal/model"
)

// Snapshot is deliberately independent of public DTOs: evidence survives removal of its target.
type ReportTarget struct {
	ID             int64  `json:"id"`
	Title          string `json:"title,omitempty"`
	Content        string `json:"content,omitempty"`
	Details        string `json:"details,omitempty"`
	Nickname       string `json:"nickname,omitempty"`
	Username       string `json:"username,omitempty"`
	Bio            string `json:"bio,omitempty"`
	Status         string `json:"status,omitempty"`
	AuthorID       int64  `json:"author_id,omitempty"`
	AuthorNickname string `json:"author_nickname,omitempty"`
	SenderID       int64  `json:"sender_id,omitempty"`
	ConversationID int64  `json:"conversation_id,omitempty"`
	UserAID        int64  `json:"user_a_id,omitempty"`
	UserBID        int64  `json:"user_b_id,omitempty"`
}

func (r *Repository) ReportTarget(ctx context.Context, kind string, id int64) (ReportTarget, error) {
	var t ReportTarget
	var err error
	switch kind {
	case "post":
		err = r.db.QueryRowContext(ctx, `SELECT p.id,p.title,p.content,p.status,p.author_id,u.nickname FROM exchange_posts p JOIN users u ON u.id=p.author_id WHERE p.id=?`, id).Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.AuthorID, &t.AuthorNickname)
	case "mate":
		err = r.db.QueryRowContext(ctx, `SELECT p.id,p.title,p.details,p.status,p.author_id,u.nickname FROM mate_posts p JOIN users u ON u.id=p.author_id WHERE p.id=?`, id).Scan(&t.ID, &t.Title, &t.Details, &t.Status, &t.AuthorID, &t.AuthorNickname)
	case "user":
		err = r.db.QueryRowContext(ctx, "SELECT id,username,nickname,bio,status FROM users WHERE id=?", id).Scan(&t.ID, &t.Username, &t.Nickname, &t.Bio, &t.Status)
		t.AuthorID = t.ID
	case "message":
		err = r.db.QueryRowContext(ctx, `SELECT m.id,m.content,m.sender_id,m.conversation_id,c.user_a_id,c.user_b_id,u.nickname FROM messages m JOIN conversations c ON c.id=m.conversation_id JOIN users u ON u.id=m.sender_id WHERE m.id=?`, id).Scan(&t.ID, &t.Content, &t.SenderID, &t.ConversationID, &t.UserAID, &t.UserBID, &t.AuthorNickname)
		t.AuthorID = t.SenderID
	default:
		err = sql.ErrNoRows
	}
	return t, err
}

func (r *Repository) CreateReport(ctx context.Context, user int64, in model.ReportInput, snapshot []byte) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		var id int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE id=? FOR UPDATE", user).Scan(&id); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM reports WHERE reporter_id=? AND target_type=? AND target_id=? AND status='pending'", user, in.TargetType, in.TargetID).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrProcessed
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO reports(reporter_id,target_type,target_id,reason,details,snapshot) VALUES (?,?,?,?,?,?)", user, in.TargetType, in.TargetID, in.Reason, in.Details, string(snapshot))
		return err
	})
}

func (r *Repository) Reports(ctx context.Context, status string) ([]model.Report, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT r.id,r.target_type,r.target_id,r.reason,r.details,r.snapshot,r.status,r.action,r.admin_note,r.created_at,r.resolved_at,u.id,u.nickname,u.username,a.nickname FROM reports r JOIN users u ON u.id=r.reporter_id LEFT JOIN users a ON a.id=r.resolved_by WHERE r.status=? ORDER BY r.created_at DESC,r.id DESC`, status)
	if err != nil {
		return nil, err
	}
	items := []model.Report{}
	for rows.Next() {
		var item model.Report
		var snapshot []byte
		var created time.Time
		var resolved sql.NullTime
		var action, note, resolver sql.NullString
		if err := rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &item.Reason, &item.Details, &snapshot, &item.Status, &action, &note, &created, &resolved, &item.Reporter.ID, &item.Reporter.Nickname, &item.Reporter.Username, &resolver); err != nil {
			rows.Close()
			return nil, err
		}
		item.Snapshot = json.RawMessage(snapshot)
		item.Action = optionalString(action)
		item.AdminNote = optionalString(note)
		item.ResolverNickname = optionalString(resolver)
		item.CreatedAt = timestamp(created)
		item.ResolvedAt = optionalTime(resolved)
		item.MessageContext = []model.MessageContext{}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].TargetType == "message" {
			context, err := r.messageContext(ctx, items[i].TargetID)
			if err != nil {
				return nil, err
			}
			items[i].MessageContext = context
		}
	}
	return items, nil
}

func (r *Repository) messageContext(ctx context.Context, id int64) ([]model.MessageContext, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT m.id,m.content,m.created_at,m.sender_id,u.nickname FROM messages m JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=(SELECT conversation_id FROM messages WHERE id=?) AND m.id>=? AND m.id<=? ORDER BY m.id`, id, max(1, id-2), id+2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.MessageContext{}
	for rows.Next() {
		var m model.MessageContext
		var created time.Time
		if err := rows.Scan(&m.ID, &m.Content, &created, &m.SenderID, &m.SenderNickname); err != nil {
			return nil, err
		}
		m.CreatedAt = timestamp(created)
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *Repository) ProcessReport(ctx context.Context, id, admin int64, in model.ReportAction) (bannedUser int64, err error) {
	err = r.transaction(ctx, func(tx *sql.Tx) error {
		var targetType, status string
		var targetID int64
		if err := tx.QueryRowContext(ctx, "SELECT target_type,target_id,status FROM reports WHERE id=? FOR UPDATE", id).Scan(&targetType, &targetID, &status); err != nil {
			return err
		}
		if status != "pending" {
			return ErrProcessed
		}
		switch in.Action {
		case "remove_content", "delete_content":
			if targetType == "post" || targetType == "mate" {
				kind := "exchange"
				if targetType == "mate" {
					kind = "mate"
				}
				query := "UPDATE " + contentTable(kind) + " SET status='removed' WHERE id=?"
				if in.Action == "delete_content" {
					query = "DELETE FROM " + contentTable(kind) + " WHERE id=?"
				}
				if _, err := tx.ExecContext(ctx, query, targetID); err != nil {
					return err
				}
			}
		case "delete_message":
			if targetType == "message" {
				if _, err := tx.ExecContext(ctx, "DELETE FROM messages WHERE id=?", targetID); err != nil {
					return err
				}
			}
		case "ban_user":
			owner := targetID
			var err error
			switch targetType {
			case "post":
				err = tx.QueryRowContext(ctx, "SELECT author_id FROM exchange_posts WHERE id=?", targetID).Scan(&owner)
			case "mate":
				err = tx.QueryRowContext(ctx, "SELECT author_id FROM mate_posts WHERE id=?", targetID).Scan(&owner)
			case "message":
				err = tx.QueryRowContext(ctx, "SELECT sender_id FROM messages WHERE id=?", targetID).Scan(&owner)
			}
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil {
				if err = banUser(ctx, tx, owner, "banned"); err != nil {
					return err
				}
				bannedUser = owner
			}
		}
		_, err := tx.ExecContext(ctx, "UPDATE reports SET status=?,action=?,admin_note=?,resolved_by=?,resolved_at=CURRENT_TIMESTAMP(3) WHERE id=?", in.Status, in.Action, in.AdminNote, admin, id)
		return err
	})
	return
}
