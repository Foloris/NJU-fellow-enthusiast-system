package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/service"
)

func (a *API) upload(w http.ResponseWriter, r *http.Request, _ model.User) error {
	const maxImage = 5 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxImage+(1<<20))
	if err := r.ParseMultipartForm(maxImage + (1 << 20)); err != nil {
		return &service.Error{Status: 400, Message: "图片不能超过5MB，请使用正确的上传格式"}
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, _, err := r.FormFile("image")
	if err != nil {
		return &service.Error{Status: 400, Message: "请选择图片"}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImage+1))
	if err != nil {
		return err
	}
	if len(data) > maxImage {
		return &service.Error{Status: 400, Message: "图片不能超过5MB"}
	}
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}
	extension, ok := extensions[http.DetectContentType(data)]
	if !ok {
		return &service.Error{Status: 400, Message: "仅支持JPG、PNG、WebP或GIF图片"}
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	name := hex.EncodeToString(random) + extension
	file, err := os.OpenFile(filepath.Join(a.cfg.UploadDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(filepath.Join(a.cfg.UploadDir, name))
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	writeJSON(w, 201, struct {
		URL string `json:"url"`
	}{"/uploads/" + name})
	return nil
}
