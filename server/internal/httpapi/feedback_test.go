package httpapi

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"campus-fellow/server/internal/config"
	"campus-fellow/server/internal/mailer"
	"campus-fellow/server/internal/realtime"
	"campus-fellow/server/internal/service"
)

type capturedFeedback struct {
	to          string
	subject     string
	body        string
	attachments []mailer.Attachment
}

type fakeFeedbackMailer struct {
	captured *capturedFeedback
	fail     bool
}

func (m *fakeFeedbackMailer) SendFeedback(_ context.Context, to, subject, body string, attachments []mailer.Attachment) error {
	if m.fail {
		return context.DeadlineExceeded
	}
	*m.captured = capturedFeedback{to: to, subject: subject, body: body, attachments: attachments}
	return nil
}

func newFeedbackHandler(t *testing.T, feedbackMailer service.FeedbackMailer) http.Handler {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>campus frontend</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	hub := realtime.New()
	t.Cleanup(hub.Close)
	s := service.New(nil, service.NewTokens("unit-test-secret-with-at-least-32-bytes"), hub,
		service.WithFeedback("feedback@example.com", feedbackMailer))
	return New(s, hub, config.Config{ClientDir: dir, UploadDir: dir})
}

func feedbackRequest(t *testing.T, kind, content string, images [][]byte) *http.Request {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("type", kind); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("content", content); err != nil {
		t.Fatal(err)
	}
	for _, image := range images {
		part, err := writer.CreateFormFile("images", "shot.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(image); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/api/feedback", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestFeedbackRouteDeliversMail(t *testing.T) {
	var captured capturedFeedback
	handler := newFeedbackHandler(t, &fakeFeedbackMailer{captured: &captured})
	image := []byte("\x89PNG\r\n\x1a\nfake-screenshot-bytes")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, feedbackRequest(t, "problem", "在发现页点击私信后页面空白，麻烦看一下", [][]byte{image}))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "反馈已提交") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if captured.to != "feedback@example.com" || captured.subject != "[问题反馈] 搭个伴" {
		t.Fatalf("unexpected envelope: %+v", captured)
	}
	if !strings.Contains(captured.body, "在发现页点击私信后页面空白") || !strings.Contains(captured.body, "问题反馈") {
		t.Fatalf("unexpected body: %s", captured.body)
	}
	if len(captured.attachments) != 1 || captured.attachments[0].Filename != "截图1.png" || captured.attachments[0].ContentType != "image/png" || !bytes.Equal(captured.attachments[0].Data, image) {
		t.Fatalf("unexpected attachment: %+v", captured.attachments)
	}
}

func TestFeedbackRouteRejectsInvalidInput(t *testing.T) {
	handler := newFeedbackHandler(t, &fakeFeedbackMailer{captured: &capturedFeedback{}})
	for _, tc := range []struct {
		name string
		req  *http.Request
	}{
		{"short-content", feedbackRequest(t, "problem", "打不开", nil)},
		{"invalid-kind", feedbackRequest(t, "other", "在发现页点击私信后页面空白", nil)},
		{"invalid-image", feedbackRequest(t, "problem", "在发现页点击私信后页面空白", [][]byte{[]byte("not-an-image")})},
		{"not-multipart", httptest.NewRequest("POST", "/api/feedback", strings.NewReader("{}"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.req)
			if w.Code != 400 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestFeedbackRouteWithoutMailer(t *testing.T) {
	handler := newFeedbackHandler(t, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, feedbackRequest(t, "suggestion", "希望能按兴趣筛选搭子", nil))
	if w.Code != 503 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
