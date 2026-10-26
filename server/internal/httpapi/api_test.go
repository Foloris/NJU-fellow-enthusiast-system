package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"campus-fellow/server/internal/config"
	"campus-fellow/server/internal/realtime"
	"campus-fellow/server/internal/service"
)

func TestRoutesAndGuestBoundaries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>campus frontend</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	hub := realtime.New()
	defer hub.Close()
	s := service.New(nil, service.NewTokens("unit-test-secret-with-at-least-32-bytes"), hub)
	handler := New(s, hub, config.Config{ClientDir: dir, UploadDir: dir})
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/api/auth/me", "", 200}, {"GET", "/api/admin/summary", "", 401}, {"POST", "/api/uploads", "", 401}, {"POST", "/api/mates/1/apply", "{}", 401}, {"GET", "/ws", "", 401},
		{"POST", "/api/auth/register", "{", 400}, {"POST", "/api/auth/register", "{} {}", 400}, {"POST", "/api/auth/register", "{}", 400},
		{"GET", "/api/unknown", "", 404}, {"GET", "/mates/123", "", 200},
	} {
		t.Run(tc.method+tc.path+tc.body, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if strings.HasPrefix(tc.path, "/api") && !json.Valid(w.Body.Bytes()) {
				t.Fatal("API response not JSON")
			}
		})
	}
}

func TestProductionCookies(t *testing.T) {
	a := &API{cfg: config.Config{Production: true}}
	w := httptest.NewRecorder()
	a.cookie(w, "token", false)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing auth cookie")
	}
	c := cookies[0]
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 604800 {
		t.Fatalf("invalid production cookie: %+v", c)
	}
	w = httptest.NewRecorder()
	a.cookie(w, "", true)
	if w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout cookie does not expire")
	}
}
