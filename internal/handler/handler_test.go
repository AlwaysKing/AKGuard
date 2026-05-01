package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

func setupTestServer(t *testing.T) (*config.AppState, *middleware.OTPStore, func()) {
	t.Helper()

	dbPath := t.TempDir() + "/test.db"
	if err := model.InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	cfg := config.LoadConfig()
	parsedNets := config.ParseCIDRs(cfg.Auth.InternalNets)

	state := &config.AppState{
		Config:     cfg,
		ParsedNets: parsedNets,
		Sessions:   config.NewSessionStore(),
		Attempts:   config.NewAttemptTracker(),
	}

	otpStore := middleware.NewOTPStore()

	cleanup := func() {
		model.DB.Close()
		os.Remove(dbPath)
	}

	return state, otpStore, cleanup
}

func verifyRequest(state *config.AppState, host, realIP string, cookie *http.Cookie) *httptest.ResponseRecorder {
	handler := Verify(state)
	req := httptest.NewRequest("GET", "/verify", nil)
	req.Host = host
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// 创建一个有效的 auth session（模拟已登录用户）
func getAuthSessionCookie(state *config.AppState, isInit bool) *http.Cookie {
	id := state.Sessions.Create("192.168.1.5", "test-agent", "auth", isInit)
	return &http.Cookie{Name: middleware.AuthCookieName, Value: id}
}

func newJSONRequest(method, path string, body interface{}) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func adminLogin(t *testing.T, state *config.AppState) *http.Cookie {
	t.Helper()
	req, _ := newJSONRequest("POST", "/api/admin/login", map[string]string{"password": "123456"})
	// httptest 的 RemoteAddr 是 "192.0.2.1:1234"，设 X-Real-IP 保持一致
	req.Header.Set("X-Real-IP", "127.0.0.1")
	rec := httptest.NewRecorder()
	otpStore := middleware.NewOTPStore()
	AdminLogin(state, otpStore).ServeHTTP(rec, req)
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == middleware.AdminCookieName {
			return c
		}
	}
	t.Fatalf("admin login failed, status=%d body=%s", rec.Code, rec.Body.String())
	return nil
}

// ========== /verify 测试 ==========

func TestVerify_Reject(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "reject", External: "reject"}

	for _, ip := range []string{"192.168.1.5", "8.8.8.8"} {
		rec := verifyRequest(state, "any.host", ip, nil)
		if rec.Code != 401 {
			t.Errorf("IP=%s: got %d, want 401", ip, rec.Code)
		}
	}
}

func TestVerify_Pass(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "pass", External: "pass"}

	for _, ip := range []string{"192.168.1.5", "8.8.8.8", ""} {
		rec := verifyRequest(state, "any.host", ip, nil)
		if rec.Code != 200 {
			t.Errorf("IP=%s: got %d, want 200", ip, rec.Code)
		}
	}
}

func TestVerify_Auth_RequireSession(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "auth", External: "auth"}

	// 无 cookie → 401
	rec := verifyRequest(state, "any.host", "192.168.1.5", nil)
	if rec.Code != 401 {
		t.Errorf("no cookie: got %d, want 401", rec.Code)
	}

	// 有效 session → 200
	cookie := getAuthSessionCookie(state, false)
	rec = verifyRequest(state, "any.host", "192.168.1.5", cookie)
	if rec.Code != 200 {
		t.Errorf("valid session: got %d, want 200", rec.Code)
	}

	// init session → 401
	initCookie := getAuthSessionCookie(state, true)
	rec = verifyRequest(state, "any.host", "192.168.1.5", initCookie)
	if rec.Code != 401 {
		t.Errorf("init session: got %d, want 401", rec.Code)
	}
}

func TestVerify_Auth_IPMismatch(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "auth", External: "auth"}

	// session 创建时 IP 为 192.168.1.5
	cookie := getAuthSessionCookie(state, false)

	// 同 IP → 200
	rec := verifyRequest(state, "any.host", "192.168.1.5", cookie)
	if rec.Code != 200 {
		t.Errorf("same IP: got %d, want 200", rec.Code)
	}

	// 不同 IP → 401
	rec = verifyRequest(state, "any.host", "8.8.8.8", cookie)
	if rec.Code != 401 {
		t.Errorf("different IP: got %d, want 401", rec.Code)
	}
}

func TestVerify_DeleteSession_InvalidatesAuth(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "auth", External: "auth"}

	cookie := getAuthSessionCookie(state, false)

	// 注销前 → 200
	rec := verifyRequest(state, "any.host", "192.168.1.5", cookie)
	if rec.Code != 200 {
		t.Errorf("before delete: got %d, want 200", rec.Code)
	}

	// 从内存删除
	state.Sessions.Delete(cookie.Value)

	// 注销后 → 401
	rec = verifyRequest(state, "any.host", "192.168.1.5", cookie)
	if rec.Code != 401 {
		t.Errorf("after delete: got %d, want 401", rec.Code)
	}
}

func TestVerify_WildcardDomain(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()

	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "reject", External: "reject"}
	state.Config.Auth.Domains = []config.DomainRule{
		{Host: "*.example.com", Policy: config.Policy{Internal: "pass", External: "auth"}},
		{Host: "api.example.com", Policy: config.Policy{Internal: "reject", External: "reject"}},
	}

	authCookie := getAuthSessionCookie(state, false)

	tests := []struct {
		name   string
		host   string
		ip     string
		cookie *http.Cookie
		expect int
	}{
		{"sub.app", "app.example.com", "192.168.1.5", nil, 200},
		{"sub.api exact first", "api.example.com", "192.168.1.5", nil, 401},
		{"sub.www", "www.example.com", "192.168.1.5", nil, 200},
		{"sub.deep", "a.b.example.com", "192.168.1.5", nil, 200},
		{"bare domain", "example.com", "192.168.1.5", nil, 401},
		{"sub external no cookie", "app.example.com", "8.8.8.8", nil, 401},
		{"sub external with cookie", "app.example.com", "192.168.1.5", authCookie, 200},
		{"sub with port", "app.example.com:443", "192.168.1.5", nil, 200},
		{"unrelated", "other.org", "192.168.1.5", nil, 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := verifyRequest(state, tt.host, tt.ip, tt.cookie)
			if rec.Code != tt.expect {
				t.Errorf("status = %d, want %d", rec.Code, tt.expect)
			}
		})
	}
}

func TestVerify_InternalVsExternal(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()

	state.Config.Auth.DefaultPolicy = config.Policy{Internal: "pass", External: "reject"}
	state.Config.Auth.InternalNets = []string{"192.168.1.0/24", "10.0.0.0/8"}
	state.ParsedNets = config.ParseCIDRs(state.Config.Auth.InternalNets)

	tests := []struct {
		ip     string
		expect int
	}{
		{"192.168.1.1", 200},
		{"10.0.0.1", 200},
		{"8.8.8.8", 401},
		{"172.16.0.1", 401},
		{"192.168.2.1", 401},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			rec := verifyRequest(state, "any.host", tt.ip, nil)
			if rec.Code != tt.expect {
				t.Errorf("got %d, want %d", rec.Code, tt.expect)
			}
		})
	}
}

// ========== API 测试 ==========

func TestConfigAPI_UpdateDefaultPolicy(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	adminCookie := adminLogin(t, state)

	req, _ := newJSONRequest("PUT", "/api/config/default-policy", map[string]string{
		"internal": "pass", "external": "reject",
	})
	req.AddCookie(adminCookie)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	rec := httptest.NewRecorder()
	UpdateDefaultPolicy(state).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if v := model.ConfigGet("default_internal_policy"); v != "pass" {
		t.Errorf("DB = %q, want 'pass'", v)
	}
}

func TestConfigAPI_InvalidCIDR(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()
	adminCookie := adminLogin(t, state)

	req, _ := newJSONRequest("PUT", "/api/config/networks", map[string]interface{}{
		"networks": []string{"not-a-cidr"},
	})
	req.AddCookie(adminCookie)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	rec := httptest.NewRecorder()
	UpdateNetworks(state).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestConfigAPI_Unauthorized(t *testing.T) {
	state, _, cleanup := setupTestServer(t)
	defer cleanup()

	req, _ := newJSONRequest("GET", "/api/config", nil)
	rec := httptest.NewRecorder()
	GetConfig(state).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("got %d, want 401", rec.Code)
	}
}
