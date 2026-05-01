package middleware

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"akguard/internal/config"

	"golang.org/x/net/publicsuffix"
)

// --- Cookie ---

const (
	AuthCookieName  = "ak_token"
	AdminCookieName = "ak_admin"
)

func GetCookieDomain(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil || host == "" {
		host = hostPort
	}
	if net.ParseIP(host) != nil {
		return host
	}
	root, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return "." + root
}

func SetCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   GetCookieDomain(r.Host),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   config.TokenExpirySeconds,
	})
}

func ClearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   GetCookieDomain(r.Host),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// --- Auth Middleware ---

// RequireAdminFunc 验证 admin session（查内存）
func RequireAdminFunc(state *config.AppState, handler func(http.ResponseWriter, *http.Request, *config.SessionEntry)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AdminCookieName)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		entry, ok := state.Sessions.Validate(cookie.Value, clientIP, "admin")
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		handler(w, r, entry)
	}
}

// --- CORS ---

func CORS(devMode bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if devMode {
				origin := r.Header.Get("Origin")
				if origin == "" {
					origin = "http://localhost:5173"
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- OTP ---

const (
	OTPExpiry   = 2 * time.Minute
	OTPCooldown = 30 * time.Second
)

type OTPEntry struct {
	Code      string
	CreatedAt time.Time
}

type OTPStore struct {
	store map[string]*OTPEntry
	mu    sync.Mutex
}

func NewOTPStore() *OTPStore {
	return &OTPStore{store: make(map[string]*OTPEntry)}
}

func (s *OTPStore) Generate() string {
	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

func (s *OTPStore) Set(otpType string, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[otpType] = &OTPEntry{Code: code, CreatedAt: time.Now()}
}

func (s *OTPStore) Verify(otpType string, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.store[otpType]
	if entry == nil || entry.Code != code || time.Since(entry.CreatedAt) > OTPExpiry {
		return false
	}
	delete(s.store, otpType)
	return true
}

func (s *OTPStore) CheckCooldown(otpType string) (bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.store[otpType]
	if entry != nil && time.Since(entry.CreatedAt) < OTPCooldown {
		retryAfter := int(OTPCooldown.Seconds() - time.Since(entry.CreatedAt).Seconds())
		return true, retryAfter
	}
	return false, 0
}

func SendBarkPush(barkURL, code string) error {
	pushURL := fmt.Sprintf("%s/AKAuth/%s", strings.TrimRight(barkURL, "/"), code)
	resp, err := http.Get(pushURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned %d", resp.StatusCode)
	}
	return nil
}

// --- JSON Response Helpers ---

func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func JSONError(w http.ResponseWriter, status int, msg string) {
	JSONResponse(w, status, map[string]string{"error": msg})
}

func JSONSuccess(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, data)
}

// --- Bark 验证码（配置时用） ---

var (
	barkVerifyCode string
	barkCodeMu     sync.Mutex
)

func GenerateBarkVerifyCode() string {
	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n)
	barkCodeMu.Lock()
	barkVerifyCode = code
	barkCodeMu.Unlock()
	return code
}

func VerifyBarkCode(code string) bool {
	barkCodeMu.Lock()
	defer barkCodeMu.Unlock()
	if barkVerifyCode == "" || barkVerifyCode != code {
		return false
	}
	barkVerifyCode = "" // 一次性
	return true
}
