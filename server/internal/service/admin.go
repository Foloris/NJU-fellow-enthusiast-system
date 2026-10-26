package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/repository"
)

func (s *Service) CreateReport(ctx context.Context, user model.User, in model.ReportInput) error {
	in.Reason = strings.TrimSpace(in.Reason)
	in.Details = strings.TrimSpace(in.Details)
	if !contains(in.TargetType, "post", "mate", "user", "message") || in.TargetID < 1 || !length(in.Reason, 2, 100) || !length(in.Details, 0, 1000) {
		return fail(400, "举报对象或举报内容不符合要求")
	}
	if in.TargetType == "user" && in.TargetID == user.ID {
		return fail(400, "不能举报自己的账号")
	}
	target, err := s.Repo.ReportTarget(ctx, in.TargetType, in.TargetID)
	if err != nil {
		return missing(err, "举报对象不存在")
	}
	if in.TargetType == "message" && target.UserAID != user.ID && target.UserBID != user.ID {
		return fail(403, "不能举报不属于你的会话中的消息")
	}
	snapshot, err := json.Marshal(target)
	if err != nil {
		return err
	}
	err = s.Repo.CreateReport(ctx, user.ID, in, snapshot)
	if errors.Is(err, repository.ErrProcessed) {
		return fail(409, "你已举报过该内容，管理员正在处理中")
	}
	return err
}
func (s *Service) SetUserStatus(ctx context.Context, admin model.User, id int64, status string) error {
	if id == admin.ID {
		return fail(400, "不能封禁当前管理员账号")
	}
	if !contains(status, "active", "banned") {
		return fail(400, "无效的账号状态")
	}
	u, err := s.Repo.UserByID(ctx, id)
	if err != nil {
		return missing(err, "用户不存在")
	}
	if u.Role == "admin" {
		return fail(400, "不能在这里修改管理员账号")
	}
	err = s.Repo.SetUserStatus(ctx, id, status)
	if err == nil && status == "banned" && s.Events != nil {
		s.Events.Disconnect(id)
	}
	return err
}
func (s *Service) SetContentStatus(ctx context.Context, kind string, id int64, status string) error {
	if !contains(kind, "exchange", "mate") {
		return fail(400, "无效的内容类型")
	}
	if !contains(status, "published", "removed") {
		return fail(400, "无效的内容状态")
	}
	return missing(s.Repo.SetContentStatus(ctx, kind, id, status), "内容不存在")
}
func (s *Service) DeleteContent(ctx context.Context, kind string, id int64) error {
	if !contains(kind, "exchange", "mate") {
		return fail(400, "无效的内容类型")
	}
	return missing(s.Repo.DeleteContent(ctx, kind, id), "内容不存在")
}
func (s *Service) ProcessReport(ctx context.Context, admin model.User, id int64, in model.ReportAction) error {
	if !contains(in.Status, "resolved", "dismissed") {
		return fail(400, "无效的处理状态")
	}
	if in.Action == "" {
		in.Action = "none"
	}
	if !contains(in.Action, "none", "remove_content", "delete_content", "ban_user", "delete_message") {
		return fail(400, "无效的处理方式")
	}
	if !length(in.AdminNote, 0, 1000) {
		return fail(400, "管理员备注不能超过1000个字符")
	}
	bannedUser, err := s.Repo.ProcessReport(ctx, id, admin.ID, in)
	if errors.Is(err, repository.ErrProcessed) {
		return fail(409, "这条举报已经处理过了")
	}
	if err == nil && bannedUser != 0 && s.Events != nil {
		s.Events.Disconnect(bannedUser)
	}
	return missing(err, "举报记录不存在")
}
