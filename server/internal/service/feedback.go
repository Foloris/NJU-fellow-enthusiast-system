package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"campus-fellow/server/internal/mailer"
)

const (
	// MaxFeedbackLength mirrors the textarea limit in the feedback dialog.
	MaxFeedbackLength = 1000
	MaxFeedbackImages = 3
)

type FeedbackInput struct {
	Kind        string
	Body        string
	Attachments []mailer.Attachment
}

// SendFeedback forwards a footer feedback form to the project mailbox. Guests may submit,
// so validation stays strict and delivery errors are reported without leaking SMTP details.
func (s *Service) SendFeedback(ctx context.Context, in FeedbackInput) error {
	kind := strings.TrimSpace(in.Kind)
	if kind != "problem" && kind != "suggestion" {
		return fail(400, "反馈类型不正确")
	}
	body := strings.TrimSpace(in.Body)
	if utf8.RuneCountInString(body) < 5 {
		return fail(400, "请至少写 5 个字，说明你遇到的情况或想法")
	}
	if utf8.RuneCountInString(body) > MaxFeedbackLength {
		return fail(400, "反馈内容不能超过" + strconv.Itoa(MaxFeedbackLength) + "字")
	}
	if len(in.Attachments) > MaxFeedbackImages {
		return fail(400, "最多只能添加" + strconv.Itoa(MaxFeedbackImages) + "张截图")
	}
	if s.feedbackMailer == nil || s.feedbackTo == "" {
		return fail(503, "反馈邮件服务尚未配置，请联系管理员")
	}
	label := "问题反馈"
	subject := "[问题反馈] 搭个伴"
	if kind == "suggestion" {
		label = "意见反馈"
		subject = "[意见反馈] 搭个伴"
	}
	lines := []string{
		"反馈类型：" + label,
		"提交时间：" + time.Now().Format("2006-01-02 15:04:05"),
		"截图数量：" + strconv.Itoa(len(in.Attachments)),
		"",
		body,
	}
	if err := s.feedbackMailer.SendFeedback(ctx, s.feedbackTo, subject, strings.Join(lines, "\r\n"), in.Attachments); err != nil {
		slog.Error("feedback mail delivery failed", "error", err)
		return fail(503, "反馈发送失败，请稍后重试")
	}
	return nil
}
