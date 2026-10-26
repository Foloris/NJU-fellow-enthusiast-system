package httpapi

import (
	"net/http"

	"campus-fellow/server/internal/model"
)

func (a *API) mateRoutes() {
	a.handle("GET /api/mates", "", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		page, err := a.service.Mates(r.Context(), user, filter(r))
		if err != nil {
			return err
		}
		writeJSON(w, 200, page)
		return nil
	})
	a.handle("GET /api/mates/{id}", "", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		post, err := a.service.Mate(r.Context(), user, id)
		if err != nil {
			return err
		}
		writeJSON(w, 200, struct {
			Post model.Mate `json:"post"`
		}{post})
		return nil
	})
	a.handle("POST /api/mates", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		in, err := decode[model.MateInput](w, r)
		if err != nil {
			return err
		}
		id, err := a.service.CreateMate(r.Context(), user, in)
		if err != nil {
			return err
		}
		writeJSON(w, 201, idResponse{ID: id})
		return nil
	})
	a.handle("PATCH /api/mates/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[model.MateInput](w, r)
		if err != nil {
			return err
		}
		if err = a.service.UpdateMate(r.Context(), user, id, in); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"搭子帖子已更新"})
		return nil
	})
	a.handle("DELETE /api/mates/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err = a.service.DeleteMate(r.Context(), user, id); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"搭子帖子已删除"})
		return nil
	})
	a.handle("POST /api/mates/{id}/apply", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[struct {
			Message string `json:"message"`
		}](w, r)
		if err != nil {
			return err
		}
		application, err := a.service.Apply(r.Context(), user, id, in.Message)
		if err != nil {
			return err
		}
		writeJSON(w, 201, idResponse{ID: application, Message: "申请已发送，等待发布者回应"})
		return nil
	})
	a.handle("GET /api/mates/{id}/applications", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		items, err := a.service.Applications(r.Context(), user, id)
		if err != nil {
			return err
		}
		writeJSON(w, 200, model.List[model.Application]{Items: items})
		return nil
	})
	a.handle("GET /api/mates/applications/mine", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		items, err := a.service.MyApplications(r.Context(), user, r.URL.Query().Get("scope") == "received")
		if err != nil {
			return err
		}
		writeJSON(w, 200, model.List[model.MyApplication]{Items: items})
		return nil
	})
	a.handle("PATCH /api/mates/applications/{id}", "user", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[struct {
			Action string `json:"action"`
		}](w, r)
		if err != nil {
			return err
		}
		result, err := a.service.RespondApplication(r.Context(), user, id, in.Action)
		if err != nil {
			return err
		}
		writeJSON(w, 200, result)
		return nil
	})
	a.handle("POST /api/mates/{id}/report", "user", a.report("mate"))
}
