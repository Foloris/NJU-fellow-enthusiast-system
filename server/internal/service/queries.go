package service

import (
	"campus-fellow/server/internal/model"
	"context"
)

func (s *Service) Health(ctx context.Context) error {
	if err := s.Repo.Ping(ctx); err != nil {
		return fail(503, "数据库暂不可用")
	}
	return nil
}
func (s *Service) MyApplications(ctx context.Context, user model.User, received bool) ([]model.MyApplication, error) {
	return s.Repo.MyApplications(ctx, user.ID, received)
}
func (s *Service) Conversations(ctx context.Context, user model.User) ([]model.Conversation, error) {
	return s.Repo.Conversations(ctx, user.ID)
}
func (s *Service) Summary(ctx context.Context) (model.Summary, error) { return s.Repo.Summary(ctx) }
func (s *Service) AdminUsers(ctx context.Context, q string) ([]model.AdminUser, error) {
	return s.Repo.AdminUsers(ctx, q)
}
func (s *Service) AdminContent(ctx context.Context, kind, q string) ([]model.AdminContent, error) {
	return s.Repo.AdminContent(ctx, kind, q)
}
func (s *Service) Reports(ctx context.Context, status string) ([]model.Report, error) {
	if !contains(status, "pending", "resolved", "dismissed") {
		status = "pending"
	}
	return s.Repo.Reports(ctx, status)
}
