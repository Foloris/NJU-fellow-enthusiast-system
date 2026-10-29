package httpapi

import (
	"fmt"
	"io"
	"net/http"

	"campus-fellow/server/internal/mailer"
	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/service"
)

var imageExtensions = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}

// feedback accepts the footer feedback form and mails it, with screenshots as attachments,
// to the project mailbox configured by FEEDBACK_EMAIL.
func (a *API) feedback(w http.ResponseWriter, r *http.Request, _ model.User) error {
	const maxImage = 5 << 20
	maxBody := int64(service.MaxFeedbackImages)*(maxImage+(1<<20)) + (1 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseMultipartForm(maxBody); err != nil {
		return &service.Error{Status: 400, Message: "反馈内容格式不正确，或图片总大小超出限制"}
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	files := r.MultipartForm.File["images"]
	if len(files) > service.MaxFeedbackImages {
		return &service.Error{Status: 400, Message: fmt.Sprintf("最多只能添加%d张截图", service.MaxFeedbackImages)}
	}
	attachments := make([]mailer.Attachment, 0, len(files))
	for index, header := range files {
		file, err := header.Open()
		if err != nil {
			return &service.Error{Status: 400, Message: "图片读取失败，请重新上传"}
		}
		data, err := io.ReadAll(io.LimitReader(file, maxImage+1))
		file.Close()
		if err != nil {
			return err
		}
		if len(data) > maxImage {
			return &service.Error{Status: 400, Message: "单张截图不能超过5MB"}
		}
		contentType := http.DetectContentType(data)
		extension, ok := imageExtensions[contentType]
		if !ok {
			return &service.Error{Status: 400, Message: "仅支持JPG、PNG、WebP或GIF图片"}
		}
		attachments = append(attachments, mailer.Attachment{
			Filename:    fmt.Sprintf("截图%d%s", index+1, extension),
			ContentType: contentType,
			Data:        data,
		})
	}
	if err := a.service.SendFeedback(r.Context(), service.FeedbackInput{
		Kind:        r.FormValue("type"),
		Body:        r.FormValue("content"),
		Attachments: attachments,
	}); err != nil {
		return err
	}
	writeJSON(w, 200, messageResponse{"反馈已提交，我们会尽快处理"})
	return nil
}
