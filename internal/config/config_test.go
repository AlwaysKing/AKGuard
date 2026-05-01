package config

import (
	"net"
	"testing"
)

func TestIsValidAction(t *testing.T) {
	tests := []struct {
		action string
		want   bool
	}{
		{"reject", true},
		{"pass", true},
		{"auth", true},
		{"allow", false},
		{"deny", false},
		{"", false},
		{"Auth", false},
	}
	for _, tt := range tests {
		if got := IsValidAction(tt.action); got != tt.want {
			t.Errorf("IsValidAction(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestParseCIDRs(t *testing.T) {
	tests := []struct {
		name   string
		cidrs  []string
		count  int
	}{
		{"valid single", []string{"192.168.1.0/24"}, 1},
		{"valid multiple", []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}, 3},
		{"with invalid", []string{"10.0.0.0/8", "invalid", "192.168.0.0/16"}, 2},
		{"empty", []string{}, 0},
		{"all invalid", []string{"not-a-cidr", "also-bad"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets := ParseCIDRs(tt.cidrs)
			if len(nets) != tt.count {
				t.Errorf("ParseCIDRs(%v) returned %d nets, want %d", tt.cidrs, len(nets), tt.count)
			}
		})
	}
}

func TestIsInternalIP(t *testing.T) {
	nets := ParseCIDRs([]string{"192.168.1.0/24", "10.0.0.0/8"})

	tests := []struct {
		ip   string
		want bool
	}{
		// 内网 IP
		{"192.168.1.1", true},
		{"192.168.1.254", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		// 外网 IP
		{"8.8.8.8", false},
		{"1.2.3.4", false},
		{"172.16.0.1", false}, // 不在配置的网段内
		// 边界
		{"192.168.1.0", true},  // 网络地址
		{"192.168.1.255", true}, // 广播地址
		{"192.168.2.1", false},  // 超出 /24
		// 无效 IP
		{"invalid", false},
		{"", false},
		// 带端口（nginx X-Real-IP 可能带端口）
		{"192.168.1.1:54321", true},
		{"10.0.0.1:8080", true},
		{"8.8.8.8:443", false},
	}
	for _, tt := range tests {
		if got := IsInternalIP(tt.ip, nets); got != tt.want {
			t.Errorf("IsInternalIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestResolvePolicy(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			DefaultPolicy: Policy{Internal: "auth", External: "reject"},
			Domains: []DomainRule{
				{Host: "app.example.com", Policy: Policy{Internal: "pass", External: "auth"}},
				{Host: "private.example.com", Policy: Policy{Internal: "pass", External: "reject"}},
				{Host: "public.example.com", Policy: Policy{Internal: "auth", External: "pass"}},
			},
		},
	}

	tests := []struct {
		name           string
		host           string
		wantInternal   string
		wantExternal   string
	}{
		// 精确匹配
		{"exact match", "app.example.com", "pass", "auth"},
		{"another exact", "private.example.com", "pass", "reject"},
		// 未配置域名 → 使用默认策略
		{"unknown domain", "unknown.example.com", "auth", "reject"},
		// 带端口 → 去掉端口后匹配
		{"with port", "app.example.com:443", "pass", "auth"},
		{"with port 80", "app.example.com:80", "pass", "auth"},
		// 无端口直接匹配
		{"no port", "public.example.com", "auth", "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := ResolvePolicy(tt.host, cfg)
			if policy.Internal != tt.wantInternal {
				t.Errorf("ResolvePolicy(%q).Internal = %q, want %q", tt.host, policy.Internal, tt.wantInternal)
			}
			if policy.External != tt.wantExternal {
				t.Errorf("ResolvePolicy(%q).External = %q, want %q", tt.host, policy.External, tt.wantExternal)
			}
		})
	}
}

func TestResolvePolicy_PortStripping(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			DefaultPolicy: Policy{Internal: "auth", External: "auth"},
			Domains: []DomainRule{
				{Host: "test.local", Policy: Policy{Internal: "pass", External: "pass"}},
			},
		},
	}

	// 验证各种端口格式都能正确去掉端口
	hosts := []string{"test.local:443", "test.local:8080", "test.local"}
	for _, h := range hosts {
		policy := ResolvePolicy(h, cfg)
		if policy.Internal != "pass" {
			t.Errorf("ResolvePolicy(%q).Internal = %q, want 'pass'", h, policy.Internal)
		}
	}
}

func TestResolvePolicy_Wildcard(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			DefaultPolicy: Policy{Internal: "reject", External: "reject"},
			Domains: []DomainRule{
				{Host: "*.example.com", Policy: Policy{Internal: "pass", External: "auth"}},
				{Host: "*.local", Policy: Policy{Internal: "pass", External: "reject"}},
				// 精确匹配优先
				{Host: "private.example.com", Policy: Policy{Internal: "reject", External: "reject"}},
			},
		},
	}

	tests := []struct {
		name         string
		host         string
		wantInternal string
		wantExternal string
	}{
		// 通配符匹配子域名
		{"sub domain", "app.example.com", "pass", "auth"},
		{"another sub", "api.example.com", "pass", "auth"},
		{"deep sub", "a.b.example.com", "pass", "auth"},

		// 通配符不匹配裸域
		{"bare domain", "example.com", "reject", "reject"},

		// 精确匹配优先于通配符
		{"exact overrides wildcard", "private.example.com", "reject", "reject"},

		// 另一个通配符规则
		{"*.local match", "test.local", "pass", "reject"},
		{"*.local no bare", "local", "reject", "reject"},

		// 不匹配的域名走默认
		{"unrelated domain", "other.org", "reject", "reject"},

		// 通配符 + 端口
		{"wildcard with port", "app.example.com:443", "pass", "auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := ResolvePolicy(tt.host, cfg)
			if policy.Internal != tt.wantInternal {
				t.Errorf("Internal = %q, want %q", policy.Internal, tt.wantInternal)
			}
			if policy.External != tt.wantExternal {
				t.Errorf("External = %q, want %q", policy.External, tt.wantExternal)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		host    string
		want    bool
	}{
		{"*.example.com", "app.example.com", true},
		{"*.example.com", "a.b.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "notexample.com", false},
		{"*.local", "test.local", true},
		{"*.local", "local", false},
		{"app.example.com", "app.example.com", false}, // 非 * 开头
		{"*", "anything", false},                        // 单独 * 不匹配
		{"", "anything", false},
		{"*.example.com", "", false},
	}
	for _, tt := range tests {
		if got := matchWildcard(tt.pattern, tt.host); got != tt.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.host, got, tt.want)
		}
	}
}

func TestHashPassword(t *testing.T) {
	password := "test123"
	hash := HashPassword(password)
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if hash == password {
		t.Fatal("HashPassword returned plaintext")
	}
}

func TestParseCIDRs_ResultFormat(t *testing.T) {
	nets := ParseCIDRs([]string{"192.168.1.0/24"})
	if len(nets) != 1 {
		t.Fatalf("expected 1 net, got %d", len(nets))
	}
	// 验证解析结果可以包含 IP
	if !nets[0].Contains(net.ParseIP("192.168.1.1")) {
		t.Fatal("192.168.1.0/24 should contain 192.168.1.1")
	}
}
