package httpapi

import (
	"net/http"

	"campus-fellow/server/internal/model"
)

func (a *API) exchangeRoutes() {
	a.handle("GET /api/exchange/posts", "", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		page, err := a.service.Exchanges(r.Context(), user, filter(r))
		if err != nil {
			return err
		}
		writeJSON(w, 200, page)
		return nil
	})
	a.handle("GET /api/exchange/posts/{id}", "", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		result, err := a.service.Exchange(r.Context(), user, id)
		if err != nil {
			return err
		}
		writeJSON(w, 200, result)
		return nil
	})
	a.handle("POST /api/exchange/posts", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		in, err := decode[model.ExchangeInput](w, r)
		if err != nil {
			return err
		}
		id, err := a.service.CreateExchange(r.Context(), user, in)
		if err != nil {
			return err
		}
		writeJSON(w, 201, idResponse{ID: id})
		return nil
	})
	a.handle("PATCH /api/exchange/posts/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[model.ExchangeInput](w, r)
		if err != nil {
			return err
		}
		if err = a.service.UpdateExchange(r.Context(), user, id, in); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"帖子已更新"})
		return nil
	})
	a.handle("DELETE /api/exchange/posts/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err = a.service.DeleteExchange(r.Context(), user, id); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"帖子已删除"})
		return nil
	})
	for _, reaction := range []string{"like", "favorite"} {
		a.handle("POST /api/exchange/posts/{id}/"+reaction, "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
			id, err := pathID(r)
			if err != nil {
				return err
			}
			result, err := a.service.React(r.Context(), user, id, reaction == "favorite")
			if err != nil {
				return err
			}
			writeJSON(w, 200, result)
			return nil
		})
	}
	a.handle("POST /api/exchange/posts/{id}/comments", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
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
		comment, err := a.service.CreateComment(r.Context(), user, id, in.Content)
		if err != nil {
			return err
		}
		writeJSON(w, 201, struct {
			Comment model.Comment `json:"comment"`
		}{comment})
		return nil
	})
	a.handle("DELETE /api/exchange/comments/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err = a.service.DeleteComment(r.Context(), user, id); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"评论已删除"})
		return nil
	})
	a.handle("POST /api/exchange/posts/{id}/report", "user", a.report("post"))
}
