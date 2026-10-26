package service

import (
	"context"
	"strings"

	"campus-fellow/server/internal/model"
)

type Event struct {
	Type           string         `json:"type"`
	ConversationID int64          `json:"conversationId,omitempty"`
	Message        *model.Message `json:"message,omitempty"`
	ReaderID       int64          `json:"readerId,omitempty"`
}
type Publisher interface {
	Publish(users []int64, event Event)
	Disconnect(user int64)
}
type MessagesResult struct {
	Items   []model.Message `json:"items"`
	HasMore bool            `json:"hasMore"`
}

func (s *Service) Messages(ctx context.Context, user model.User, id, before int64, limit int) (MessagesResult, error) {
	if _, err := s.Repo.Conversation(ctx, id, user.ID); err != nil {
		return MessagesResult{}, missing(err, "会话不存在")
	}
	items, err := s.Repo.Messages(ctx, id, before, limit)
	return MessagesResult{Items: items, HasMore: len(items) == limit}, err
}
func (s *Service) SendMessage(ctx context.Context, user model.User, id int64, content string) (model.Message, error) {
	c, err := s.Repo.Conversation(ctx, id, user.ID)
	if err != nil {
		return model.Message{}, missing(err, "会话不存在")
	}
	content = strings.TrimSpace(content)
	if !length(content, 1, 2000) {
		return model.Message{}, fail(400, "消息内容应为1—2000个字符")
	}
	message, err := s.Repo.CreateMessage(ctx, id, user.ID, content)
	if err != nil {
		return message, err
	}
	if s.Events != nil {
		s.Events.Publish([]int64{c.UserAID, c.UserBID}, Event{Type: "message", ConversationID: id, Message: &message})
	}
	return message, nil
}
func (s *Service) ReadMessages(ctx context.Context, user model.User, id int64) (int64, error) {
	c, err := s.Repo.Conversation(ctx, id, user.ID)
	if err != nil {
		return 0, missing(err, "会话不存在")
	}
	updated, err := s.Repo.ReadMessages(ctx, id, user.ID)
	if err == nil && s.Events != nil {
		s.Events.Publish([]int64{c.UserAID, c.UserBID}, Event{Type: "read", ConversationID: id, ReaderID: user.ID})
	}
	return updated, err
}
