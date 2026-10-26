package service

import (
	"context"
	"errors"
	"strings"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/repository"
)

func protectContact(p *model.Mate, user model.User) {
	p.IsOwner = user.ID != 0 && user.ID == p.AuthorID
	if !canManage(user, p.AuthorID) && (p.MyApplicationStatus == nil || *p.MyApplicationStatus != "accepted") {
		p.Contact = nil
		hint := "申请通过后可查看联系方式并私聊"
		p.ContactHint = &hint
	}
}
func (s *Service) Mates(ctx context.Context, user model.User, f model.Filter) (model.Page[model.Mate], error) {
	if f.Mine && user.ID == 0 {
		return model.Page[model.Mate]{}, fail(401, "请先登录")
	}
	page, err := s.Repo.Mates(ctx, user.ID, f)
	for i := range page.Items {
		protectContact(&page.Items[i], user)
	}
	return page, err
}
func (s *Service) Mate(ctx context.Context, user model.User, id int64) (model.Mate, error) {
	p, err := s.Repo.Mate(ctx, id, user.ID)
	if err != nil {
		return p, missing(err, "搭子帖子不存在")
	}
	if p.Status != "published" && !canManage(user, p.AuthorID) {
		return model.Mate{}, fail(404, "搭子帖子不存在或已被下架")
	}
	protectContact(&p, user)
	return p, nil
}
func (s *Service) CreateMate(ctx context.Context, user model.User, in model.MateInput) (int64, error) {
	if err := validateMate(&in); err != nil {
		return 0, err
	}
	return s.Repo.CreateMate(ctx, user.ID, in)
}
func (s *Service) UpdateMate(ctx context.Context, user model.User, id int64, in model.MateInput) error {
	p, err := s.Repo.Mate(ctx, id, user.ID)
	if err != nil {
		return missing(err, "搭子帖子不存在")
	}
	if !canManage(user, p.AuthorID) {
		return fail(403, "只能修改自己发布的搭子")
	}
	if err := validateMate(&in); err != nil {
		return err
	}
	return s.Repo.UpdateMate(ctx, id, in)
}
func (s *Service) DeleteMate(ctx context.Context, user model.User, id int64) error {
	p, err := s.Repo.Mate(ctx, id, user.ID)
	if err != nil {
		return missing(err, "搭子帖子不存在")
	}
	if !canManage(user, p.AuthorID) {
		return fail(403, "只能删除自己发布的搭子")
	}
	return s.Repo.DeleteMate(ctx, id)
}
func (s *Service) Apply(ctx context.Context, user model.User, id int64, message string) (int64, error) {
	message = strings.TrimSpace(message)
	if !length(message, 1, 500) {
		return 0, fail(400, "申请说明应为1—500个字符")
	}
	p, err := s.Repo.Mate(ctx, id, user.ID)
	if err != nil {
		return 0, missing(err, "搭子帖子不存在或已停止招募")
	}
	if p.Status != "published" {
		return 0, fail(404, "搭子帖子不存在或已停止招募")
	}
	if p.AuthorID == user.ID {
		return 0, fail(400, "不能申请自己发布的搭子")
	}
	application, err := s.Repo.Apply(ctx, id, user.ID, message)
	if repository.Duplicate(err) {
		return 0, fail(409, "你已经申请过这个搭子")
	}
	return application, err
}
func (s *Service) Applications(ctx context.Context, user model.User, id int64) ([]model.Application, error) {
	p, err := s.Repo.Mate(ctx, id, user.ID)
	if err != nil {
		return nil, missing(err, "搭子帖子不存在")
	}
	if !canManage(user, p.AuthorID) {
		return nil, fail(403, "只有发布者可以查看申请")
	}
	return s.Repo.Applications(ctx, id)
}

type ApplicationResult struct {
	Message        string `json:"message"`
	ConversationID *int64 `json:"conversationId"`
}

func (s *Service) RespondApplication(ctx context.Context, user model.User, id int64, action string) (ApplicationResult, error) {
	if !contains(action, "accept", "reject") {
		return ApplicationResult{}, fail(400, "无效的处理方式")
	}
	a, err := s.Repo.Application(ctx, id)
	if err != nil {
		return ApplicationResult{}, missing(err, "申请不存在")
	}
	if !canManage(user, a.OwnerID) {
		return ApplicationResult{}, fail(403, "只有发布者可以处理申请")
	}
	conversation, err := s.Repo.RespondApplication(ctx, a, action == "accept")
	if errors.Is(err, repository.ErrProcessed) {
		return ApplicationResult{}, fail(409, "这条申请已经处理过了")
	}
	message := "已拒绝申请"
	if action == "accept" {
		message = "已同意申请，可以开始私聊"
	}
	return ApplicationResult{Message: message, ConversationID: conversation}, err
}
