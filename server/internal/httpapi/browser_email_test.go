package httpapi

import (
	"html/template"
	"net/http"
	"sort"
	"sync"
)

// Test-only UI for manually exercising the actual frontend without external mail.
// This handler is compiled exclusively into go test and listens on httptest's loopback socket.
func emailBrowserFixture(app http.Handler, mailbox *testMailbox) (http.Handler, <-chan struct{}) {
	done := make(chan struct{})
	var once sync.Once
	page := template.Must(template.New("mailbox").Parse(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>隔离测试收件箱</title><body><h1>隔离测试收件箱</h1><p>仅用于本机临时数据库测试，没有发送真实邮件。</p><a href="/__test/mailbox">刷新收件箱</a>{{range .}}<article><h2>{{.Email}}</h2><p>验证码：<strong>{{.Code}}</strong></p></article>{{else}}<p>暂无测试邮件</p>{{end}}<form method="post" action="/__test/finish"><button>结束浏览器验证</button></form></body></html>`))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__test/mailbox", func(w http.ResponseWriter, r *http.Request) {
		type delivery struct{ Email, Code string }
		items := []delivery{}
		mailbox.mu.Lock()
		for email, code := range mailbox.codes {
			items = append(items, delivery{email, code})
		}
		mailbox.mu.Unlock()
		sort.Slice(items, func(i, j int) bool { return items[i].Email < items[j].Email })
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = page.Execute(w, items)
	})
	mux.HandleFunc("POST /__test/finish", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("测试结束，临时库将自动清理。"))
		once.Do(func() { close(done) })
	})
	mux.Handle("/", app)
	return mux, done
}
