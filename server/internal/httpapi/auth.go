package httpapi

import (
	"net"
	"net/http"
	"time"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/service"
)

func (a *API) routes() {
	a.handle("GET /api/auth/registration-config", "", func(w http.ResponseWriter, _ *http.Request, _ model.User) error {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, a.service.RegistrationSettings())
		return nil
	})
	a.handle("POST /api/auth/email-code", "", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		in, err := decode[service.EmailCodeInput](w, r)
		if err != nil {
			return err
		}
		source, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			source = r.RemoteAddr
		}
		result, err := a.service.SendEmailCode(r.Context(), in.Email, source)
		if err != nil {
			return err
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, result)
		return nil
	})
	a.handle("GET /api/health", "", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		if err := a.service.Health(r.Context()); err != nil {
			return err
		}
		writeJSON(w, 200, struct {
			OK   bool   `json:"ok"`
			Time string `json:"time"`
		}{true, time.Now().UTC().Format(time.RFC3339Nano)})
		return nil
	})
	a.handle("POST /api/auth/register", "", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		in, err := decode[service.RegisterInput](w, r)
		if err != nil {
			return err
		}
		user, err := a.service.Register(r.Context(), in)
		if err != nil {
			return err
		}
		return a.session(w, user, 201)
	})
	a.handle("POST /api/auth/login", "", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		in, err := decode[service.LoginInput](w, r)
		if err != nil {
			return err
		}
		user, err := a.service.Login(r.Context(), in)
		if err != nil {
			return err
		}
		return a.session(w, user, 200)
	})
	a.handle("POST /api/auth/logout", "", func(w http.ResponseWriter, _ *http.Request, _ model.User) error {
		a.cookie(w, "", true)
		writeJSON(w, 200, messageResponse{"已退出登录"})
		return nil
	})
	a.handle("GET /api/auth/me", "", func(w http.ResponseWriter, _ *http.Request, user model.User) error {
		var current *model.User
		if user.ID != 0 {
			current = &user
		}
		writeJSON(w, 200, userResponse{User: current})
		return nil
	})
	a.handle("PATCH /api/auth/me", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		in, err := decode[service.ProfileInput](w, r)
		if err != nil {
			return err
		}
		updated, err := a.service.UpdateProfile(r.Context(), user, in)
		if err != nil {
			return err
		}
		writeJSON(w, 200, userResponse{User: &updated})
		return nil
	})
	a.handle("GET /api/users/{id}", "", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		user, err := a.service.PublicUser(r.Context(), id)
		if err != nil {
			return err
		}
		writeJSON(w, 200, struct {
			User service.PublicUser `json:"user"`
		}{user})
		return nil
	})
	a.handle("GET /ws", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		// Upgrade writes its own HTTP error response; do not write JSON after hijacking.
		_ = a.hub.Serve(w, r, user.ID)
		return nil
	})
	a.handle("POST /api/uploads", "user", a.upload)
	a.handle("POST /api/feedback", "", a.feedback)
	a.exchangeRoutes()
	a.mateRoutes()
	a.messageRoutes()
	a.adminRoutes()
}
