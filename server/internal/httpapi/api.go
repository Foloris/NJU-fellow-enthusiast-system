package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"campus-fellow/server/internal/config"
	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/realtime"
	"campus-fellow/server/internal/service"
)

type API struct {
	service *service.Service
	hub     *realtime.Hub
	cfg     config.Config
	mux     *http.ServeMux
}
type endpoint func(http.ResponseWriter, *http.Request, model.User) error
type messageResponse struct {
	Message string `json:"message"`
}
type idResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message,omitempty"`
}
type userResponse struct {
	User *model.User `json:"user"`
}

func New(s *service.Service, hub *realtime.Hub, cfg config.Config) http.Handler {
	a := &API{service: s, hub: hub, cfg: cfg, mux: http.NewServeMux()}
	a.routes()
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	a.mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		uploads.ServeHTTP(w, r)
	}))
	a.mux.HandleFunc("/", a.frontend)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("request panic", "path", r.URL.Path, "panic", recovered)
				writeJSON(w, 500, messageResponse{Message: "服务器发生错误"})
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}
		a.mux.ServeHTTP(w, r)
	})
}

func (a *API) handle(pattern, role string, fn endpoint) {
	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		raw := ""
		if cookie, err := r.Cookie("campus_token"); err == nil {
			raw = cookie.Value
		}
		if raw == "" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			raw = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		u, err := a.service.Authenticate(r.Context(), raw)
		if err != nil {
			writeError(w, r, err)
			return
		}
		user := model.User{}
		if u != nil {
			user = *u
		}
		if role != "" && u == nil {
			writeError(w, r, &service.Error{Status: 401, Message: "请先登录后再继续"})
			return
		}
		if role == "admin" && user.Role != "admin" {
			writeError(w, r, &service.Error{Status: 403, Message: "仅管理员可以访问"})
			return
		}
		if err := fn(w, r, user); err != nil {
			writeError(w, r, err)
		}
	})
}

func writeJSON[T any](w http.ResponseWriter, status int, value T) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var e *service.Error
	if errors.As(err, &e) {
		if e.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(e.RetryAfter))
		}
		writeJSON(w, e.Status, messageResponse{Message: e.Message})
		return
	}
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeJSON(w, 500, messageResponse{Message: "服务器发生错误，请稍后重试"})
}
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var value T
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&value); err != nil {
		return value, &service.Error{Status: 400, Message: "请求内容格式不正确或超过1MB"}
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return value, &service.Error{Status: 400, Message: "请求只能包含一个JSON对象"}
	}
	return value, nil
}
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &service.Error{Status: 400, Message: "无效的记录编号"}
	}
	return id, nil
}
func number(value string, fallback, maximum int) int {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 {
		return fallback
	}
	return min(v, maximum)
}
func filter(r *http.Request) model.Filter {
	q := r.URL.Query()
	return model.Filter{Query: strings.TrimSpace(q.Get("q")), Kind: q.Get("kind"), Tag: strings.TrimSpace(q.Get("tag")), Mine: q.Get("mine") == "true", Page: number(q.Get("page"), 1, 1000000), PageSize: number(q.Get("pageSize"), 12, 50)}
}
func (a *API) cookie(w http.ResponseWriter, value string, clear bool) {
	cookie := &http.Cookie{Name: "campus_token", Value: value, Path: "/", HttpOnly: true, Secure: a.cfg.Production, SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 60 * 60}
	if clear {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, cookie)
}
func (a *API) session(w http.ResponseWriter, user model.User, status int) error {
	token, err := a.service.Tokens.Sign(user)
	if err != nil {
		return err
	}
	a.cookie(w, token, false)
	writeJSON(w, status, userResponse{User: &user})
	return nil
}
func (a *API) frontend(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/uploads/") || r.URL.Path == "/ws" {
		writeJSON(w, 404, messageResponse{Message: "接口不存在"})
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		writeJSON(w, 405, messageResponse{Message: "不支持的请求方法"})
		return
	}
	clean := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	file := filepath.Join(a.cfg.ClientDir, filepath.FromSlash(clean))
	info, err := os.Stat(file)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, file)
		return
	}
	index := filepath.Join(a.cfg.ClientDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		writeJSON(w, 503, messageResponse{Message: "前端尚未构建，请运行 npm run build 或使用 Vite 开发服务"})
		return
	}
	http.ServeFile(w, r, index)
}
