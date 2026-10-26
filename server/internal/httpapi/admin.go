package httpapi

import (
	"net/http"
	"strings"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/service"
)

func (a *API) report(kind string) endpoint {
	return func(w http.ResponseWriter, r *http.Request, user model.User) error {
		in, err := decode[model.ReportInput](w, r)
		if err != nil {
			return err
		}
		if kind != "" {
			id, err := pathID(r)
			if err != nil {
				return err
			}
			in.TargetType = kind
			in.TargetID = id
		} else if in.TargetType == "post" || in.TargetType == "mate" {
			return &service.Error{Status: 400, Message: "请从对应的帖子页面提交举报"}
		}
		if err = a.service.CreateReport(r.Context(), user, in); err != nil {
			return err
		}
		writeJSON(w, 201, messageResponse{"举报已提交，管理员会尽快处理"})
		return nil
	}
}

func (a *API) adminRoutes() {
	a.handle("POST /api/reports", "user", a.report(""))
	a.handle("GET /api/reports/admin", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		items, err := a.service.Reports(r.Context(), r.URL.Query().Get("status"))
		if err != nil {
			return err
		}
		writeJSON(w, 200, model.List[model.Report]{Items: items})
		return nil
	})
	a.handle("GET /api/admin/summary", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		summary, err := a.service.Summary(r.Context())
		if err != nil {
			return err
		}
		writeJSON(w, 200, summary)
		return nil
	})
	a.handle("GET /api/admin/users", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		items, err := a.service.AdminUsers(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")))
		if err != nil {
			return err
		}
		writeJSON(w, 200, model.List[model.AdminUser]{Items: items})
		return nil
	})
	a.handle("PATCH /api/admin/users/{id}", "admin", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[struct {
			Status string `json:"status"`
		}](w, r)
		if err != nil {
			return err
		}
		if err = a.service.SetUserStatus(r.Context(), user, id, in.Status); err != nil {
			return err
		}
		message := "账号已解除封禁"
		if in.Status == "banned" {
			message = "账号已封禁"
		}
		writeJSON(w, 200, messageResponse{message})
		return nil
	})
	a.handle("GET /api/admin/content", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		kind := "exchange"
		if r.URL.Query().Get("type") == "mate" {
			kind = "mate"
		}
		items, err := a.service.AdminContent(r.Context(), kind, strings.TrimSpace(r.URL.Query().Get("q")))
		if err != nil {
			return err
		}
		writeJSON(w, 200, struct {
			Type  string               `json:"type"`
			Items []model.AdminContent `json:"items"`
		}{kind, items})
		return nil
	})
	a.handle("PATCH /api/admin/content/{type}/{id}", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[struct {
			Status string `json:"status"`
		}](w, r)
		if err != nil {
			return err
		}
		if err = a.service.SetContentStatus(r.Context(), r.PathValue("type"), id, in.Status); err != nil {
			return err
		}
		message := "内容已恢复"
		if in.Status == "removed" {
			message = "内容已下架"
		}
		writeJSON(w, 200, messageResponse{message})
		return nil
	})
	a.handle("DELETE /api/admin/content/{type}/{id}", "admin", func(w http.ResponseWriter, r *http.Request, _ model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err = a.service.DeleteContent(r.Context(), r.PathValue("type"), id); err != nil {
			return err
		}
		writeJSON(w, 200, messageResponse{"内容已删除"})
		return nil
	})
	a.handle("PATCH /api/admin/reports/{id}", "admin", func(w http.ResponseWriter, r *http.Request, user model.User) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		in, err := decode[model.ReportAction](w, r)
		if err != nil {
			return err
		}
		if err = a.service.ProcessReport(r.Context(), user, id, in); err != nil {
			return err
		}
		message := "举报已处理"
		if in.Status == "dismissed" {
			message = "举报已驳回"
		}
		writeJSON(w, 200, messageResponse{message})
		return nil
	})
}
