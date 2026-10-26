package httpapi

import (
	"net/http"
	"strconv"

	"campus-fellow/server/internal/model"
)

func (a *API) messageRoutes() {
	a.handle("GET /api/messages", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		items, err := a.service.Conversations(r.Context(), user)
		if err != nil {
			return err
		}
		writeJSON(w, 200, model.List[model.Conversation]{Items: items})
		return nil
	})
	a.handle("GET /api/messages/{id}/messages", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		result, err := a.service.Messages(r.Context(), user, id, max(0, before), number(r.URL.Query().Get("limit"), 50, 100))
		if err != nil {
			return err
		}
		writeJSON(w, 200, result)
		return nil
	})
	a.handle("POST /api/messages/{id}/messages", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[struct {
			Content string `json:"content"`
		}](w, r)
		if err != nil {
			return err
		}
		message, err := a.service.SendMessage(r.Context(), user, id, in.Content)
		if err != nil {
			return err
		}
		writeJSON(w, 201, struct {
			Message model.Message `json:"message"`
		}{message})
		return nil
	})
	a.handle("POST /api/messages/{id}/read", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		updated, err := a.service.ReadMessages(r.Context(), user, id)
		if err != nil {
			return err
		}
		writeJSON(w, 200, struct {
			Updated int64 `json:"updated"`
		}{updated})
		return nil
	})
}
