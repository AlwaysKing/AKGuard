package config

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"sync"
	"time"
)

// SessionEntry 内存中的会话记录
type SessionEntry struct {
	ID        string    `json:"id"`
	ClientIP  string    `json:"client_ip"`
	UserAgent string    `json:"user_agent"`
	TokenType string    `json:"token_type"` // "auth" or "admin"
	IsInit    bool      `json:"is_init"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore 基于内存的会话存储
type SessionStore struct {
	sessions map[string]*SessionEntry
	mu       sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*SessionEntry),
	}
}

// generateID 生成随机会话 ID
func generateID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Create 创建新会话，返回 session ID
func (s *SessionStore) Create(clientIP, userAgent, tokenType string, isInit bool) string {
	id := generateID()
	entry := &SessionEntry{
		ID:        id,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		TokenType: tokenType,
		IsInit:    isInit,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(TokenExpirySeconds * time.Second),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = entry
	return id
}

// Get 获取会话（不存在或已过期返回 nil）
func (s *SessionStore) Get(id string) *SessionEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.sessions[id]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry
}

// Delete 删除会话
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// DeleteByType 删除某类型的所有会话
func (s *SessionStore) DeleteByType(tokenType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, entry := range s.sessions {
		if entry.TokenType == tokenType {
			delete(s.sessions, id)
		}
	}
}

// List 列出所有有效会话（同时清理过期的）
func (s *SessionStore) List() []*SessionEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	result := make([]*SessionEntry, 0, len(s.sessions))
	for id, entry := range s.sessions {
		if now.After(entry.ExpiresAt) {
			delete(s.sessions, id)
			continue
		}
		result = append(result, entry)
	}
	return result
}

// Count 返回有效会话数量
func (s *SessionStore) Count() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var count int64
	for id, entry := range s.sessions {
		if now.After(entry.ExpiresAt) {
			delete(s.sessions, id)
			continue
		}
		count++
	}
	return count
}

// Validate 验证会话：存在 + 未过期 + 类型匹配 + IP 匹配
func (s *SessionStore) Validate(id, clientIP, expectedType string) (*SessionEntry, bool) {
	entry := s.Get(id)
	if entry == nil {
		return nil, false
	}
	if entry.TokenType != expectedType {
		return nil, false
	}
	if !ipEqual(entry.ClientIP, clientIP) {
		return nil, false
	}
	return entry, true
}

// ipEqual 比较两个 IP（去掉端口后比较）
func ipEqual(a, b string) bool {
	aIP, _, err := net.SplitHostPort(a)
	if err != nil {
		aIP = a
	}
	bIP, _, err := net.SplitHostPort(b)
	if err != nil {
		bIP = b
	}
	return aIP == bIP
}

// AttemptTracker 内存中的登录失败记录
type AttemptTracker struct {
	mu       sync.RWMutex
	attempts map[string][]time.Time // key: "auth:1.2.3.4" or "admin:1.2.3.4"
}

func NewAttemptTracker() *AttemptTracker {
	return &AttemptTracker{
		attempts: make(map[string][]time.Time),
	}
}

// Record 记录一次失败，返回窗口内的失败次数
func (t *AttemptTracker) Record(banType, ip string, windowSec int) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := banType + ":" + stripPortFromIP(ip)
	now := time.Now()
	cutoff := now.Add(-time.Duration(windowSec) * time.Second)

	// 过滤掉过期的记录
	recent := make([]time.Time, 0, len(t.attempts[key]))
	for _, at := range t.attempts[key] {
		if at.After(cutoff) {
			recent = append(recent, at)
		}
	}
	recent = append(recent, now)
	t.attempts[key] = recent

	return len(recent)
}

// Clear 登录成功后清除计数
func (t *AttemptTracker) Clear(banType, ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := banType + ":" + stripPortFromIP(ip)
	delete(t.attempts, key)
}

func stripPortFromIP(s string) string {
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		return s
	}
	return host
}
