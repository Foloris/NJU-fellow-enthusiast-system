package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"campus-fellow/server/internal/model"
)

var ErrProcessed = errors.New("record already processed")

const mateSelect = `SELECT p.id,p.author_id,p.title,p.tags,p.details,p.image_url,p.location,p.contact,p.duration,p.notes,p.status,p.created_at,p.updated_at,
 (SELECT COUNT(*) FROM applications a WHERE a.mate_post_id=p.id),
 (SELECT a.status FROM applications a WHERE a.mate_post_id=p.id AND a.applicant_id=?),
 u.id,u.username,u.nickname,u.avatar FROM mate_posts p JOIN users u ON u.id=p.author_id `

func scanMate(row scanner) (model.Mate, error) {
	var p model.Mate
	var tags string
	var image, location, notes, status, avatar sql.NullString
	var created, updated time.Time
	var contact string
	err := row.Scan(&p.ID, &p.AuthorID, &p.Title, &tags, &p.Details, &image, &location, &contact, &p.Duration, &notes, &p.Status, &created, &updated, &p.ApplicationCount, &status, &p.Author.ID, &p.Author.Username, &p.Author.Nickname, &avatar)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal([]byte(tags), &p.Tags); err != nil {
		return p, err
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	p.ImageURL = optionalString(image)
	p.Location = optionalString(location)
	p.Notes = optionalString(notes)
	p.MyApplicationStatus = optionalString(status)
	p.Author.Avatar = optionalString(avatar)
	p.Contact = &contact
	p.CreatedAt = timestamp(created)
	p.UpdatedAt = timestamp(updated)
	return p, nil
}

func (r *Repository) Mates(ctx context.Context, viewer int64, f model.Filter) (model.Page[model.Mate], error) {
	conditions := []string{}
	args := []any{}
	if f.Mine {
		conditions = append(conditions, "p.author_id=?")
		args = append(args, viewer)
	} else {
		conditions = append(conditions, "p.status='published'", "u.status='active'")
	}
	if f.Query != "" {
		conditions = append(conditions, "(p.title LIKE ? OR p.details LIKE ? OR CAST(p.tags AS CHAR) LIKE ? OR p.location LIKE ?)")
		for range 4 {
			args = append(args, "%"+f.Query+"%")
		}
	}
	if f.Tag != "" {
		conditions = append(conditions, "JSON_CONTAINS(p.tags, JSON_QUOTE(?))")
		args = append(args, f.Tag)
	}
	where := "WHERE " + strings.Join(conditions, " AND ")
	page := model.Page[model.Mate]{Items: []model.Mate{}, Page: f.Page, PageSize: f.PageSize}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mate_posts p JOIN users u ON u.id=p.author_id "+where, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	params := append([]any{viewer}, args...)
	params = append(params, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.QueryContext(ctx, mateSelect+where+" ORDER BY p.created_at DESC,p.id DESC LIMIT ? OFFSET ?", params...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanMate(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, p)
	}
	return page, rows.Err()
}
func (r *Repository) Mate(ctx context.Context, id, viewer int64) (model.Mate, error) {
	return scanMate(r.db.QueryRowContext(ctx, mateSelect+" WHERE p.id=?", viewer, id))
}
func (r *Repository) CreateMate(ctx context.Context, author int64, in model.MateInput) (int64, error) {
	tags, err := json.Marshal(in.Tags)
	if err != nil {
		return 0, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO mate_posts(author_id,title,tags,details,image_url,location,contact,duration,notes) VALUES (?,?,?,?,?,?,?,?,?)`, author, in.Title, string(tags), in.Details, in.ImageURL, in.Location, in.Contact, in.Duration, in.Notes)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (r *Repository) UpdateMate(ctx context.Context, id int64, in model.MateInput) error {
	tags, err := json.Marshal(in.Tags)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE mate_posts SET title=?,tags=?,details=?,image_url=?,location=?,contact=?,duration=?,notes=?,updated_at=CURRENT_TIMESTAMP(3) WHERE id=?`, in.Title, string(tags), in.Details, in.ImageURL, in.Location, in.Contact, in.Duration, in.Notes, id)
	return err
}
func (r *Repository) DeleteMate(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM mate_posts WHERE id=?", id)
	return err
}
func (r *Repository) Apply(ctx context.Context, post, user int64, message string) (int64, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO applications(mate_post_id,applicant_id,message) VALUES (?,?,?)", post, user, message)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) Application(ctx context.Context, id int64) (model.Application, error) {
	var a model.Application
	err := r.db.QueryRowContext(ctx, `SELECT a.id,a.mate_post_id,a.applicant_id,a.status,p.author_id FROM applications a JOIN mate_posts p ON p.id=a.mate_post_id WHERE a.id=?`, id).Scan(&a.ID, &a.MateID, &a.ApplicantID, &a.Status, &a.OwnerID)
	return a, err
}

func (r *Repository) Applications(ctx context.Context, post int64) ([]model.Application, error) {
	items := []model.Application{}
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.message,a.status,a.created_at,a.responded_at,c.id,u.id,u.username,u.nickname,u.avatar,u.bio FROM applications a JOIN users u ON u.id=a.applicant_id LEFT JOIN conversations c ON c.application_id=a.id WHERE a.mate_post_id=? ORDER BY CASE a.status WHEN 'pending' THEN 0 ELSE 1 END,a.created_at DESC,a.id DESC`, post)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a model.Application
		var created time.Time
		var responded sql.NullTime
		var conversation sql.NullInt64
		var avatar sql.NullString
		if err := rows.Scan(&a.ID, &a.Message, &a.Status, &created, &responded, &conversation, &a.Applicant.ID, &a.Applicant.Username, &a.Applicant.Nickname, &avatar, &a.Applicant.Bio); err != nil {
			return nil, err
		}
		a.CreatedAt = timestamp(created)
		a.RespondedAt = optionalTime(responded)
		if conversation.Valid {
			a.ConversationID = &conversation.Int64
		}
		a.Applicant.Avatar = optionalString(avatar)
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *Repository) MyApplications(ctx context.Context, user int64, received bool) ([]model.MyApplication, error) {
	condition := "a.applicant_id=?"
	if received {
		condition = "p.author_id=?"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.message,a.status,a.created_at,p.id,p.title,p.status,requester.nickname,owner.nickname,c.id FROM applications a JOIN mate_posts p ON p.id=a.mate_post_id JOIN users requester ON requester.id=a.applicant_id JOIN users owner ON owner.id=p.author_id LEFT JOIN conversations c ON c.application_id=a.id WHERE `+condition+" ORDER BY a.created_at DESC,a.id DESC", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.MyApplication{}
	for rows.Next() {
		var a model.MyApplication
		var created time.Time
		var conversation sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Message, &a.Status, &created, &a.Post.ID, &a.Post.Title, &a.Post.Status, &a.ApplicantNickname, &a.OwnerNickname, &conversation); err != nil {
			return nil, err
		}
		a.CreatedAt = timestamp(created)
		if conversation.Valid {
			a.ConversationID = &conversation.Int64
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *Repository) RespondApplication(ctx context.Context, a model.Application, accept bool) (conversation *int64, err error) {
	err = r.transaction(ctx, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, "SELECT status FROM applications WHERE id=? FOR UPDATE", a.ID).Scan(&status); err != nil {
			return err
		}
		if status != "pending" {
			return ErrProcessed
		}
		next := "rejected"
		if accept {
			next = "accepted"
		}
		if _, err := tx.ExecContext(ctx, "UPDATE applications SET status=?,responded_at=CURRENT_TIMESTAMP(3) WHERE id=?", next, a.ID); err != nil {
			return err
		}
		if accept {
			result, err := tx.ExecContext(ctx, "INSERT INTO conversations(application_id,user_a_id,user_b_id) VALUES (?,?,?)", a.ID, a.OwnerID, a.ApplicantID)
			if err != nil {
				return err
			}
			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			conversation = &id
		}
		return nil
	})
	return
}
