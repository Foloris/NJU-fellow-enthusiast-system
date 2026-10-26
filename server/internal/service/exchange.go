package service

import (
	"context"
	"strings"
	"time"

	"campus-fellow/server/internal/model"
)

func canManage(user model.User, owner int64) bool {
	return user.Role == "admin" || user.ID != 0 && user.ID == owner
}
func (s *Service) Exchanges(ctx context.Context, user model.User, f model.Filter) (model.Page[model.Exchange], error) {
	if f.Mine && user.ID == 0 {
		return model.Page[model.Exchange]{}, fail(401, "请先登录")
	}
	return s.Repo.Exchanges(ctx, user.ID, f)
}

type ExchangeDetail struct {
	Post     model.Exchange  `json:"post"`
	Comments []model.Comment `json:"comments"`
}

func (s *Service) Exchange(ctx context.Context, user model.User, id int64) (ExchangeDetail, error) {
	p, err := s.Repo.Exchange(ctx, id, user.ID)
	if err != nil {
		return ExchangeDetail{}, missing(err, "帖子不存在")
	}
	if p.Status != "published" && !canManage(user, p.AuthorID) {
		return ExchangeDetail{}, fail(404, "帖子不存在或已被下架")
	}
	comments, err := s.Repo.Comments(ctx, id)
	return ExchangeDetail{Post: p, Comments: comments}, err
}
func (s *Service) CreateExchange(ctx context.Context, user model.User, in model.ExchangeInput) (int64, error) {
	if err := validateExchange(&in); err != nil {
		return 0, err
	}
	if in.Kind == "news" && user.Role != "admin" {
		return 0, fail(403, "只有管理员可以发布校园新闻")
	}
	return s.Repo.CreateExchange(ctx, user.ID, in)
}
func (s *Service) UpdateExchange(ctx context.Context, user model.User, id int64, in model.ExchangeInput) error {
	p, err := s.Repo.Exchange(ctx, id, user.ID)
	if err != nil {
		return missing(err, "帖子不存在")
	}
	if !canManage(user, p.AuthorID) {
		return fail(403, "只能修改自己发布的帖子")
	}
	if err := validateExchange(&in); err != nil {
		return err
	}
	if in.Kind == "news" && user.Role != "admin" {
		return fail(403, "只有管理员可以发布校园新闻")
	}
	return s.Repo.UpdateExchange(ctx, id, in)
}
func (s *Service) DeleteExchange(ctx context.Context, user model.User, id int64) error {
	p, err := s.Repo.Exchange(ctx, id, user.ID)
	if err != nil {
		return missing(err, "帖子不存在")
	}
	if !canManage(user, p.AuthorID) {
		return fail(403, "只能删除自己发布的帖子")
	}
	return s.Repo.DeleteExchange(ctx, id)
}

type Reaction struct {
	Active bool `json:"active"`
	Count  int  `json:"count"`
}

func (s *Service) React(ctx context.Context, user model.User, id int64, favorite bool) (Reaction, error) {
	active, count, err := s.Repo.ToggleReaction(ctx, id, user.ID, favorite)
	return Reaction{Active: active, Count: count}, missing(err, "帖子不存在或已被下架")
}
func (s *Service) CreateComment(ctx context.Context, user model.User, id int64, content string) (model.Comment, error) {
	content = strings.TrimSpace(content)
	if !length(content, 1, 500) {
		return model.Comment{}, fail(400, "评论内容应为1—500个字符")
	}
	p, err := s.Repo.Exchange(ctx, id, user.ID)
	if err != nil {
		return model.Comment{}, missing(err, "帖子不存在或已被下架")
	}
	if p.Status != "published" {
		return model.Comment{}, fail(404, "帖子不存在或已被下架")
	}
	commentID, err := s.Repo.CreateComment(ctx, id, user.ID, content)
	return model.Comment{ID: commentID, Content: content, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Author: model.Author{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Avatar: user.Avatar}}, err
}
func (s *Service) DeleteComment(ctx context.Context, user model.User, id int64) error {
	owner, err := s.Repo.CommentOwner(ctx, id)
	if err != nil {
		return missing(err, "评论不存在")
	}
	if !canManage(user, owner) {
		return fail(403, "不能删除这条评论")
	}
	return s.Repo.DeleteComment(ctx, id)
}
