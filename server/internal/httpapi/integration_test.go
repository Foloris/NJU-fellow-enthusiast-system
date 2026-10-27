package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"campus-fellow/server/internal/config"
	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/realtime"
	"campus-fellow/server/internal/repository"
	"campus-fellow/server/internal/service"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

var mysqlTests = flag.Bool("mysql", false, "run integration tests in a temporary MySQL database")
var browserEmailTests = flag.Bool("browser-email", false, "serve real frontend with a disposable email test inbox; requires -mysql")

type testClient struct {
	client *http.Client
	base   string
}

type testMailbox struct {
	mu    sync.Mutex
	codes map[string]string
	fail  bool
}

func (m *testMailbox) SendVerification(_ context.Context, email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return fmt.Errorf("simulated SMTP failure")
	}
	if m.codes == nil {
		m.codes = make(map[string]string)
	}
	m.codes[email] = code
	return nil
}
func (m *testMailbox) code(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codes[email]
}
func (m *testMailbox) setFailure(fail bool) { m.mu.Lock(); defer m.mu.Unlock(); m.fail = fail }

func registerVerified(t *testing.T, c testClient, mailbox *testMailbox, in service.RegisterInput, status int) userResponse {
	t.Helper()
	if in.Email == "" {
		in.Email = strings.ToLower(in.Username) + "@test.campus.example"
	}
	call[service.EmailCodeResult](t, c, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: in.Email}, 200)
	in.EmailCode = mailbox.code(in.Email)
	return call[userResponse](t, c, "POST", "/api/auth/register", in, status)
}

func newClient(base string) testClient {
	jar, _ := cookiejar.New(nil)
	return testClient{client: &http.Client{Jar: jar, Timeout: 10 * time.Second}, base: base}
}
func (c testClient) request(method, path string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	return response.StatusCode, data, err
}
func call[T any](t *testing.T, c testClient, method, path string, body any, want int) T {
	t.Helper()
	status, data, err := c.request(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	if status != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, status, want, data)
	}
	var result T
	if err = json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode %s: %v (%s)", path, err, data)
	}
	return result
}
func pathFor(pattern string, id int64) string { return fmt.Sprintf(pattern, id) }
func socketFor(t *testing.T, c testClient) *websocket.Conn {
	t.Helper()
	u, _ := url.Parse(c.base)
	cookies := c.client.Jar.Cookies(u)
	parts := []string{}
	for _, cookie := range cookies {
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(c.base, "http")+"/ws", http.Header{"Cookie": {strings.Join(parts, "; ")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.Close() })
	var connected service.Event
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err := ws.ReadJSON(&connected); err != nil || connected.Type != "connected" {
		t.Fatalf("socket greeting: %+v %v", connected, err)
	}
	return ws
}
func event(t *testing.T, ws *websocket.Conn, kind string) service.Event {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	var value service.Event
	if err := ws.ReadJSON(&value); err != nil {
		t.Fatal(err)
	}
	if value.Type != kind {
		t.Fatalf("event=%s want=%s", value.Type, kind)
	}
	return value
}

func TestIntegration(t *testing.T) {
	if !*mysqlTests {
		t.Skip("requires independent MySQL test database: npm run smoke")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("configure root .env first: %v", err)
	}
	// Always create our own schema; DB_NAME is never used for integration writes.
	random := make([]byte, 5)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	prefix := "it" + hex.EncodeToString(random)
	testDatabase := "campus_" + prefix + "_test"
	mc := mysql.NewConfig()
	mc.User, mc.Passwd, mc.Net, mc.Addr = cfg.DBUser, cfg.DBPassword, "tcp", net.JoinHostPort(cfg.DBHost, cfg.DBPort)
	connector, err := mysql.NewConnector(mc)
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	db.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE DATABASE `" + testDatabase + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatalf("cannot create isolated test database (account needs CREATE/DROP DATABASE): %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DROP DATABASE `" + testDatabase + "`"); err != nil {
			t.Errorf("cleanup temporary database %s: %v", testDatabase, err)
		}
	})
	cfg.DBName = testDatabase
	cfg.Production = false
	cfg.SeedDemo = false
	cfg.UploadDir = t.TempDir()
	cfg.ClientDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(cfg.ClientDir, "index.html"), []byte("<html>test SPA</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if *browserEmailTests {
		cfg.ClientDir = filepath.Join(root, "client/dist")
	}
	ctx := context.Background()
	repo, err := repository.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migration not repeatable: %v", err)
	}
	mc.DBName = testDatabase
	dataConnector, err := mysql.NewConnector(mc)
	if err != nil {
		t.Fatal(err)
	}
	testDB := sql.OpenDB(dataConnector)
	t.Cleanup(func() { testDB.Close() })
	// Verify fresh initialization and repeat initialization before exercising APIs.
	seedCfg := cfg
	seedCfg.SeedDemo = true
	for range 2 {
		if err := repo.Seed(ctx, seedCfg); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	var seededUsers, seededPosts, seededMates int
	if err := testDB.QueryRow("SELECT (SELECT COUNT(*) FROM users),(SELECT COUNT(*) FROM exchange_posts),(SELECT COUNT(*) FROM mate_posts)").Scan(&seededUsers, &seededPosts, &seededMates); err != nil {
		t.Fatal(err)
	}
	if seededUsers != 4 || seededPosts != 3 || seededMates != 3 {
		t.Fatalf("seed not idempotent: users=%d posts=%d mates=%d", seededUsers, seededPosts, seededMates)
	}
	password := "Integration123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec("INSERT INTO users(username,password_hash,nickname,role) VALUES (?,?,?,'admin')", prefix+"_admin", string(hash), "测试管理员"); err != nil {
		t.Fatal(err)
	}
	hub := realtime.New()
	mailbox := &testMailbox{}
	app := service.New(repo, service.NewTokens("isolated-integration-test-jwt-key-with-32-bytes"), hub, service.WithEmailVerification("test.campus.example", mailbox))
	handler := New(app, hub, cfg)
	var browserDone <-chan struct{}
	if *browserEmailTests {
		handler, browserDone = emailBrowserFixture(handler, mailbox)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(func() { hub.Close(); server.Close() })
	if *browserEmailTests {
		t.Logf("BROWSER_EMAIL_URL=%s (test inbox: %s/__test/mailbox)", server.URL, server.URL)
		select {
		case <-browserDone:
			return
		case <-time.After(10 * time.Minute):
			t.Fatal("browser email fixture timed out")
		}
	}
	guest, alice, bob, outsider, admin := newClient(server.URL), newClient(server.URL), newClient(server.URL), newClient(server.URL), newClient(server.URL)
	call[userResponse](t, admin, "POST", "/api/auth/login", service.LoginInput{Username: prefix + "_admin", Password: password}, 200)
	aliceUser := registerVerified(t, alice, mailbox, service.RegisterInput{Username: prefix + "_a", Password: password, Nickname: "测试甲😀"}, 201)
	bobUser := registerVerified(t, bob, mailbox, service.RegisterInput{Username: prefix + "_b", Password: password, Nickname: "测试乙"}, 201)
	registerVerified(t, outsider, mailbox, service.RegisterInput{Username: prefix + "_c", Password: password, Nickname: "无关用户"}, 201)
	t.Run("email verification policy and single use", func(t *testing.T) {
		settings := call[service.RegistrationSettings](t, guest, "GET", "/api/auth/registration-config", nil, 200)
		if settings.EmailDomain != "test.campus.example" || settings.CodeLifetimeSeconds != 60 || !settings.EmailDeliveryAvailable {
			t.Fatalf("registration settings: %+v", settings)
		}
		for _, email := range []string{"not-an-email", "student@other.example", "student@sub.test.campus.example", "Name <student@test.campus.example>"} {
			call[messageResponse](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 400)
		}
		call[messageResponse](t, guest, "POST", "/api/auth/register", service.RegisterInput{Username: prefix + "_no_code", Password: password, Nickname: "缺少验证码", Email: prefix + "_none@test.campus.example"}, 400)
		email := prefix + "_verify@test.campus.example"
		result := call[service.EmailCodeResult](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: strings.ToUpper(email)}, 200)
		if result.ExpiresIn != 60 || result.RetryAfter != 60 {
			t.Fatal("code must expire after 60 seconds")
		}
		call[messageResponse](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 429)
		in := service.RegisterInput{Username: prefix + "_verify", Password: password, Nickname: "邮箱验证", Email: email, EmailCode: "000000"}
		if mailbox.code(email) == in.EmailCode {
			in.EmailCode = "999999"
		}
		call[messageResponse](t, guest, "POST", "/api/auth/register", in, 400)
		in.EmailCode = mailbox.code(email)
		if _, err := testDB.Exec("UPDATE email_verifications SET expires_at=DATE_SUB(UTC_TIMESTAMP(3),INTERVAL 1 SECOND) WHERE email=?", email); err != nil {
			t.Fatal(err)
		}
		call[messageResponse](t, guest, "POST", "/api/auth/register", in, 400)
		if _, err := testDB.Exec("UPDATE email_verifications SET next_send_at=DATE_SUB(UTC_TIMESTAMP(3),INTERVAL 1 SECOND) WHERE email=?", email); err != nil {
			t.Fatal(err)
		}
		call[service.EmailCodeResult](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 200)
		in.EmailCode = mailbox.code(email)
		created := call[userResponse](t, guest, "POST", "/api/auth/register", in, 201)
		if created.User.Email == nil || *created.User.Email != email {
			t.Fatal("verified mailbox not bound to account")
		}
		var stored string
		var validSeconds int
		if err := testDB.QueryRow("SELECT code_hash,TIMESTAMPDIFF(SECOND,UTC_TIMESTAMP(3),expires_at) FROM email_verifications WHERE email=?", email).Scan(&stored, &validSeconds); err != nil {
			t.Fatal(err)
		}
		if stored == in.EmailCode || len(stored) != 64 || validSeconds > 60 {
			t.Fatal("code must be stored as digest with 60 second expiry")
		}
		in.Username = prefix + "_reuse"
		call[messageResponse](t, guest, "POST", "/api/auth/register", in, 400)
		call[messageResponse](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 409)
		call[messageResponse](t, guest, "POST", "/api/auth/logout", nil, 200)
	})
	t.Run("wrong code attempts and rollback", func(t *testing.T) {
		email := prefix + "_attempts@test.campus.example"
		call[service.EmailCodeResult](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 200)
		in := service.RegisterInput{Username: prefix + "_attempts", Password: password, Nickname: "错误次数", Email: email, EmailCode: "000000"}
		if in.EmailCode == mailbox.code(email) {
			in.EmailCode = "999999"
		}
		for i := 0; i < 5; i++ {
			status := 400
			if i == 4 {
				status = 429
			}
			call[messageResponse](t, guest, "POST", "/api/auth/register", in, status)
		}
		in.EmailCode = mailbox.code(email)
		call[messageResponse](t, guest, "POST", "/api/auth/register", in, 429)
		var count int
		if err := testDB.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", in.Username).Scan(&count); err != nil || count != 0 {
			t.Fatal("invalid codes created a user")
		}
		duplicateEmail := prefix + "_dup@test.campus.example"
		call[service.EmailCodeResult](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: duplicateEmail}, 200)
		duplicate := service.RegisterInput{Username: prefix + "_a", Password: password, Nickname: "重复账号", Email: duplicateEmail, EmailCode: mailbox.code(duplicateEmail)}
		call[messageResponse](t, guest, "POST", "/api/auth/register", duplicate, 409)
		duplicate.Username = prefix + "_rollback"
		call[userResponse](t, guest, "POST", "/api/auth/register", duplicate, 201)
		call[messageResponse](t, guest, "POST", "/api/auth/logout", nil, 200)
	})
	t.Run("SMTP failure cannot activate a code", func(t *testing.T) {
		email := prefix + "_smtp@test.campus.example"
		mailbox.setFailure(true)
		call[messageResponse](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 503)
		mailbox.setFailure(false)
		var count int
		if err := testDB.QueryRow("SELECT COUNT(*) FROM email_verifications WHERE email=?", email).Scan(&count); err != nil || count != 0 {
			t.Fatal("SMTP failure left a usable verification")
		}
		call[service.EmailCodeResult](t, guest, "POST", "/api/auth/email-code", service.EmailCodeInput{Email: email}, 200)
	})
	t.Run("authentication and authorization", func(t *testing.T) {
		if call[userResponse](t, guest, "GET", "/api/auth/me", nil, 200).User != nil {
			t.Fatal("guest has user")
		}
		call[messageResponse](t, guest, "POST", "/api/exchange/posts", model.ExchangeInput{}, 401)
		call[messageResponse](t, bob, "GET", "/api/admin/summary", nil, 403)
		registerVerified(t, guest, mailbox, service.RegisterInput{Username: strings.ToUpper(prefix + "_a"), Password: password, Nickname: "重复账号", Email: prefix + "_case@test.campus.example"}, 409)
		call[messageResponse](t, guest, "POST", "/api/auth/login", service.LoginInput{Username: prefix + "_a", Password: "wrong"}, 401)
		call[userResponse](t, guest, "POST", "/api/auth/login", service.LoginInput{Username: strings.ToUpper(prefix + "_a"), Password: password}, 200)
		call[messageResponse](t, guest, "POST", "/api/auth/logout", nil, 200)
		if call[userResponse](t, guest, "GET", "/api/auth/me", nil, 200).User != nil {
			t.Fatal("logout failed")
		}
		call[userResponse](t, alice, "PATCH", "/api/auth/me", service.ProfileInput{Nickname: "更新昵称😀", Bio: "中文资料与emoji😀"}, 200)
		public := call[struct {
			User service.PublicUser `json:"user"`
		}](t, guest, "GET", pathFor("/api/users/%d", aliceUser.User.ID), nil, 200)
		if public.User.Nickname != "更新昵称😀" {
			t.Fatal("profile update not persisted")
		}
	})
	post := call[idResponse](t, alice, "POST", "/api/exchange/posts", model.ExchangeInput{Title: prefix + "分享", Content: "这是用于验证完整分享业务流程的正文。"}, 201).ID
	postURL := pathFor("/api/exchange/posts/%d", post)
	t.Run("exchange CRUD search reactions comments", func(t *testing.T) {
		in := model.ExchangeInput{Kind: "share", Title: prefix + "更新分享", Content: "更新后的正文足够长，也包含表情😀。"}
		call[messageResponse](t, bob, "PATCH", postURL, in, 403)
		call[messageResponse](t, bob, "DELETE", postURL, nil, 403)
		call[messageResponse](t, alice, "PATCH", postURL, in, 200)
		call[messageResponse](t, bob, "POST", "/api/exchange/posts", model.ExchangeInput{Kind: "news", Title: "普通用户新闻", Content: "普通用户应当无法发布校园新闻内容。"}, 403)
		call[idResponse](t, admin, "POST", "/api/exchange/posts", model.ExchangeInput{Kind: "news", Title: prefix + "校园新闻", Content: "管理员发布校园新闻的正文用于验收。"}, 201)
		page := call[model.Page[model.Exchange]](t, guest, "GET", "/api/exchange/posts?q="+prefix+"&pageSize=1", nil, 200)
		if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Kind != "news" {
			t.Fatalf("search/news order: %+v", page)
		}
		call[messageResponse](t, guest, "GET", "/api/exchange/posts?mine=true", nil, 401)
		for _, kind := range []string{"like", "favorite"} {
			on := call[service.Reaction](t, bob, "POST", postURL+"/"+kind, nil, 200)
			off := call[service.Reaction](t, bob, "POST", postURL+"/"+kind, nil, 200)
			if !on.Active || on.Count != 1 || off.Active || off.Count != 0 {
				t.Fatal("reaction toggle failed")
			}
		}
		comment := call[struct {
			Comment model.Comment `json:"comment"`
		}](t, bob, "POST", postURL+"/comments", struct {
			Content string `json:"content"`
		}{"评论😀"}, 201)
		call[messageResponse](t, alice, "DELETE", pathFor("/api/exchange/comments/%d", comment.Comment.ID), nil, 403)
		detail := call[service.ExchangeDetail](t, guest, "GET", postURL, nil, 200)
		if len(detail.Comments) != 1 || detail.Comments[0].Author.ID != bobUser.User.ID {
			t.Fatal("comment response incompatible")
		}
		call[messageResponse](t, bob, "DELETE", pathFor("/api/exchange/comments/%d", comment.Comment.ID), nil, 200)
		call[messageResponse](t, bob, "POST", postURL+"/comments", struct {
			Content string `json:"content"`
		}{strings.Repeat("字", 501)}, 400)
	})
	t.Run("concurrent toggles", func(t *testing.T) {
		results := make(chan error, 4)
		for range 4 {
			go func() {
				status, _, err := bob.request("POST", postURL+"/like", nil)
				if err == nil && status != 200 {
					err = fmt.Errorf("toggle status %d", status)
				}
				results <- err
			}()
		}
		for range 4 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		if call[service.ExchangeDetail](t, bob, "GET", postURL, nil, 200).Post.LikeCount != 0 {
			t.Fatal("concurrent toggles lost an update")
		}
	})
	t.Run("upload validation and static image", func(t *testing.T) {
		png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a50sAAAAASUVORK5CYII=")
		for _, tc := range []struct {
			name string
			data []byte
			want int
		}{{"valid.png", png, 201}, {"spoof.png", []byte("<html>not an image</html>"), 400}, {"large.png", append(png, make([]byte, 5<<20)...), 400}} {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("image", tc.name)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(tc.data)
			writer.Close()
			req, _ := http.NewRequest("POST", server.URL+"/api/uploads", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			response, err := alice.client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != tc.want {
				t.Fatalf("upload %s: %d %s", tc.name, response.StatusCode, data)
			}
			if tc.want == 201 {
				var result struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatal(err)
				}
				res, err := guest.client.Get(server.URL + result.URL)
				if err != nil {
					t.Fatal(err)
				}
				uploaded, _ := io.ReadAll(res.Body)
				res.Body.Close()
				if !bytes.Equal(uploaded, png) {
					t.Fatal("uploaded bytes changed")
				}
			}
		}
	})
	mateIn := model.MateInput{Title: prefix + "周末搭子", Tags: []string{"户外", "周末"}, Details: "一起去校园公共场所完成测试活动。", Location: ptr("中心广场"), Contact: "仅通过后可见的联系信息", Duration: "周六", Notes: ptr("带水")}
	mate := call[idResponse](t, alice, "POST", "/api/mates", mateIn, 201).ID
	mateURL := pathFor("/api/mates/%d", mate)
	var conversation int64
	t.Run("mate privacy applications and concurrent approval", func(t *testing.T) {
		for _, c := range []testClient{guest, bob} {
			detail := call[struct {
				Post model.Mate `json:"post"`
			}](t, c, "GET", mateURL, nil, 200)
			if detail.Post.Contact != nil {
				t.Fatal("contact leaked before approval")
			}
		}
		owner := call[struct {
			Post model.Mate `json:"post"`
		}](t, alice, "GET", mateURL, nil, 200)
		if owner.Post.Contact == nil || !owner.Post.IsOwner {
			t.Fatal("owner cannot see contact")
		}
		call[messageResponse](t, bob, "PATCH", mateURL, mateIn, 403)
		mateIn.Notes = ptr("更新备注")
		call[messageResponse](t, alice, "PATCH", mateURL, mateIn, 200)
		matches := call[model.Page[model.Mate]](t, guest, "GET", "/api/mates?q="+prefix+"&tag="+url.QueryEscape("户外"), nil, 200)
		if matches.Total != 1 {
			t.Fatal("tag filter failed")
		}
		call[messageResponse](t, alice, "POST", mateURL+"/apply", struct {
			Message string `json:"message"`
		}{"自己申请"}, 400)
		application := call[idResponse](t, bob, "POST", mateURL+"/apply", struct {
			Message string `json:"message"`
		}{"想参加😀"}, 201).ID
		call[messageResponse](t, bob, "POST", mateURL+"/apply", struct {
			Message string `json:"message"`
		}{"重复申请"}, 409)
		call[messageResponse](t, bob, "GET", mateURL+"/applications", nil, 403)
		call[messageResponse](t, outsider, "PATCH", pathFor("/api/mates/applications/%d", application), struct {
			Action string `json:"action"`
		}{"accept"}, 403)
		rejected := call[idResponse](t, outsider, "POST", mateURL+"/apply", struct {
			Message string `json:"message"`
		}{"拒绝流程测试"}, 201).ID
		result := call[service.ApplicationResult](t, alice, "PATCH", pathFor("/api/mates/applications/%d", rejected), struct {
			Action string `json:"action"`
		}{"reject"}, 200)
		if result.ConversationID != nil {
			t.Fatal("rejection created a conversation")
		}
		type approval struct {
			status int
			data   []byte
			err    error
		}
		results := make(chan approval, 2)
		for range 2 {
			go func() {
				status, data, err := alice.request("PATCH", pathFor("/api/mates/applications/%d", application), struct {
					Action string `json:"action"`
				}{"accept"})
				results <- approval{status, data, err}
			}()
		}
		statuses := map[int]int{}
		for range 2 {
			result := <-results
			if result.err != nil {
				t.Fatal(result.err)
			}
			statuses[result.status]++
			if result.status == 200 {
				var accepted service.ApplicationResult
				if err := json.Unmarshal(result.data, &accepted); err != nil {
					t.Fatal(err)
				}
				if accepted.ConversationID == nil {
					t.Fatal("missing conversation")
				}
				conversation = *accepted.ConversationID
			}
		}
		if statuses[200] != 1 || statuses[409] != 1 {
			t.Fatalf("approval statuses: %v", statuses)
		}
		var count int
		if err := testDB.QueryRow("SELECT COUNT(*) FROM conversations WHERE application_id=?", application).Scan(&count); err != nil || count != 1 {
			t.Fatalf("conversation count=%d err=%v", count, err)
		}
		accepted := call[struct {
			Post model.Mate `json:"post"`
		}](t, bob, "GET", mateURL, nil, 200)
		if accepted.Post.Contact == nil || *accepted.Post.Contact != mateIn.Contact {
			t.Fatal("contact not revealed after approval")
		}
		for _, scope := range []struct {
			client testClient
			name   string
		}{{bob, "sent"}, {alice, "received"}} {
			items := call[model.List[model.MyApplication]](t, scope.client, "GET", "/api/mates/applications/mine?scope="+scope.name, nil, 200)
			if len(items.Items) == 0 {
				t.Fatal("application history empty")
			}
		}
	})
	var messageID int64
	t.Run("realtime history unread and isolation", func(t *testing.T) {
		messageURL := pathFor("/api/messages/%d/messages", conversation)
		call[messageResponse](t, outsider, "GET", messageURL, nil, 404)
		call[messageResponse](t, outsider, "POST", messageURL, struct {
			Content string `json:"content"`
		}{"越权消息"}, 404)
		wsA, wsB := socketFor(t, alice), socketFor(t, bob)
		sent := call[struct {
			Message model.Message `json:"message"`
		}](t, bob, "POST", messageURL, struct {
			Content string `json:"content"`
		}{"实时私信😀"}, 201)
		messageID = sent.Message.ID
		for _, ws := range []*websocket.Conn{wsA, wsB} {
			pushed := event(t, ws, "message")
			if pushed.Message == nil || pushed.Message.ID != messageID || pushed.ConversationID != conversation {
				t.Fatal("realtime payload mismatch")
			}
		}
		conversations := call[model.List[model.Conversation]](t, alice, "GET", "/api/messages", nil, 200)
		if len(conversations.Items) != 1 || conversations.Items[0].UnreadCount != 1 {
			t.Fatalf("unread count: %+v", conversations)
		}
		call[struct {
			Updated int `json:"updated"`
		}](t, alice, "POST", pathFor("/api/messages/%d/read", conversation), nil, 200)
		event(t, wsA, "read")
		event(t, wsB, "read")
		history := call[service.MessagesResult](t, alice, "GET", messageURL+"?limit=1", nil, 200)
		if len(history.Items) != 1 || history.Items[0].ReadAt == nil || !history.HasMore {
			t.Fatal("message history/read receipt mismatch")
		}
		earlier := call[service.MessagesResult](t, alice, "GET", messageURL+fmt.Sprintf("?before=%d", messageID), nil, 200)
		if len(earlier.Items) != 0 {
			t.Fatal("history cursor failed")
		}
		_, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
		if err == nil {
			t.Fatal("guest websocket accepted")
		}
		if response == nil || response.StatusCode != 401 {
			t.Fatal("guest websocket not rejected with 401")
		}
		response.Body.Close()
	})
	t.Run("report evidence moderation and persistence", func(t *testing.T) {
		call[messageResponse](t, bob, "POST", postURL+"/report", model.ReportInput{Reason: "内容举报", Details: "自动化验收"}, 201)
		call[messageResponse](t, bob, "POST", postURL+"/report", model.ReportInput{Reason: "重复举报"}, 409)
		call[messageResponse](t, bob, "POST", mateURL+"/report", model.ReportInput{Reason: "搭子举报"}, 201)
		call[messageResponse](t, alice, "POST", "/api/reports", model.ReportInput{TargetType: "user", TargetID: bobUser.User.ID, Reason: "账号举报"}, 201)
		call[messageResponse](t, outsider, "POST", "/api/reports", model.ReportInput{TargetType: "message", TargetID: messageID, Reason: "越权举报"}, 403)
		call[messageResponse](t, alice, "POST", "/api/reports", model.ReportInput{TargetType: "message", TargetID: messageID, Reason: "私信举报"}, 201)
		reports := call[model.List[model.Report]](t, admin, "GET", "/api/reports/admin?status=pending", nil, 200)
		var postReport, messageReport, mateReport, userReport int64
		for _, report := range reports.Items {
			switch {
			case report.TargetType == "post" && report.TargetID == post:
				postReport = report.ID
			case report.TargetType == "message" && report.TargetID == messageID:
				messageReport = report.ID
				if len(report.MessageContext) == 0 || report.MessageContext[0].SenderID != bobUser.User.ID {
					t.Fatal("missing typed message context")
				}
			case report.TargetType == "mate" && report.TargetID == mate:
				mateReport = report.ID
			case report.TargetType == "user" && report.TargetID == bobUser.User.ID:
				userReport = report.ID
			}
		}
		if postReport == 0 || messageReport == 0 || mateReport == 0 || userReport == 0 {
			t.Fatal("reports missing")
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", postReport), model.ReportAction{Status: "resolved", Action: "remove_content", AdminNote: "测试下架"}, 200)
		call[messageResponse](t, guest, "GET", postURL, nil, 404)
		call[service.ExchangeDetail](t, alice, "GET", postURL, nil, 200)
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", postReport), model.ReportAction{Status: "resolved", Action: "none"}, 409)
		for range 2 {
			call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/content/exchange/%d", post), struct {
				Status string `json:"status"`
			}{"published"}, 200)
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", messageReport), model.ReportAction{Status: "resolved", Action: "delete_message"}, 200)
		if len(call[service.MessagesResult](t, alice, "GET", pathFor("/api/messages/%d/messages", conversation), nil, 200).Items) != 0 {
			t.Fatal("moderated message still exists")
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", mateReport), model.ReportAction{Status: "dismissed", Action: "none"}, 200)
		bannedSocket := socketFor(t, bob)
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", userReport), model.ReportAction{Status: "resolved", Action: "ban_user"}, 200)
		if call[userResponse](t, bob, "GET", "/api/auth/me", nil, 200).User != nil {
			t.Fatal("report ban did not revoke active session")
		}
		bannedSocket.SetReadDeadline(time.Now().Add(3 * time.Second))
		if _, _, err := bannedSocket.ReadMessage(); err == nil {
			t.Fatal("report ban left socket open")
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/users/%d", bobUser.User.ID), struct {
			Status string `json:"status"`
		}{"active"}, 200)
		call[model.Summary](t, admin, "GET", "/api/admin/summary", nil, 200)
		users := call[model.List[model.AdminUser]](t, admin, "GET", "/api/admin/users?q="+prefix, nil, 200)
		if len(users.Items) != 6 {
			t.Fatal("admin user search mismatch")
		}
		call[struct {
			Items []model.AdminContent `json:"items"`
		}](t, admin, "GET", "/api/admin/content?type=mate&q="+prefix, nil, 200)
		second, err := repository.Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer second.Close()
		if err := second.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := second.Exchange(ctx, post, 0); err != nil {
			t.Fatalf("MySQL persistence: %v", err)
		}
	})
	t.Run("content deletion through report retains evidence", func(t *testing.T) {
		id := call[idResponse](t, bob, "POST", "/api/exchange/posts", model.ExchangeInput{Title: prefix + "删除测试", Content: "举报处理删除内容后仍然应保留审核证据。"}, 201).ID
		call[messageResponse](t, alice, "POST", pathFor("/api/exchange/posts/%d/report", id), model.ReportInput{Reason: "删除测试"}, 201)
		reports := call[model.List[model.Report]](t, admin, "GET", "/api/reports/admin?status=pending", nil, 200)
		var reportID int64
		for _, item := range reports.Items {
			if item.TargetType == "post" && item.TargetID == id {
				reportID = item.ID
			}
		}
		if reportID == 0 {
			t.Fatal("missing deletion report")
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/reports/%d", reportID), model.ReportAction{Status: "resolved", Action: "delete_content"}, 200)
		call[messageResponse](t, guest, "GET", pathFor("/api/exchange/posts/%d", id), nil, 404)
		resolved := call[model.List[model.Report]](t, admin, "GET", "/api/reports/admin?status=resolved", nil, 200)
		found := false
		for _, item := range resolved.Items {
			if item.ID == reportID {
				found = bytes.Contains(item.Snapshot, []byte("删除测试"))
			}
		}
		if !found {
			t.Fatal("deleted content lost report evidence")
		}
		disposable := call[idResponse](t, bob, "POST", "/api/mates", mateIn, 201).ID
		call[messageResponse](t, outsider, "DELETE", pathFor("/api/mates/%d", disposable), nil, 403)
		call[messageResponse](t, bob, "DELETE", pathFor("/api/mates/%d", disposable), nil, 200)
	})
	t.Run("banning sessions and cascade deletion", func(t *testing.T) {
		ws := socketFor(t, alice)
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/users/%d", aliceUser.User.ID), struct {
			Status string `json:"status"`
		}{"banned"}, 200)
		if call[userResponse](t, alice, "GET", "/api/auth/me", nil, 200).User != nil {
			t.Fatal("banned session still active")
		}
		call[messageResponse](t, alice, "POST", "/api/exchange/posts", model.ExchangeInput{}, 401)
		call[messageResponse](t, guest, "POST", "/api/auth/login", service.LoginInput{Username: prefix + "_a", Password: password}, 403)
		call[messageResponse](t, guest, "GET", postURL, nil, 404)
		ws.SetReadDeadline(time.Now().Add(3 * time.Second))
		if _, _, err := ws.ReadMessage(); err == nil {
			t.Fatal("banned socket still open")
		}
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/users/%d", aliceUser.User.ID), struct {
			Status string `json:"status"`
		}{"active"}, 200)
		call[messageResponse](t, guest, "GET", postURL, nil, 404)
		call[messageResponse](t, admin, "PATCH", pathFor("/api/admin/content/exchange/%d", post), struct {
			Status string `json:"status"`
		}{"published"}, 200)
		call[messageResponse](t, admin, "DELETE", pathFor("/api/admin/content/mate/%d", mate), nil, 200)
		call[messageResponse](t, bob, "GET", pathFor("/api/messages/%d/messages", conversation), nil, 404)
		call[messageResponse](t, alice, "DELETE", postURL, nil, 200)
		call[messageResponse](t, guest, "GET", postURL, nil, 404)
	})
}

func ptr(s string) *string { return &s }
