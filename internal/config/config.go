package config

import (
	"log"
	"net"

	"akguard/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// --- 配置结构 ---

type Policy struct {
	Internal string `json:"internal"`
	External string `json:"external"`
}

type DomainRule struct {
	Host   string `json:"host"`
	Policy `json:",inline"`
}

type BanConfig struct {
	Enabled     bool `json:"enabled"`
	WindowSec   int  `json:"window_sec"`   // 统计窗口（秒）
	MaxAttempts int  `json:"max_attempts"` // 最大失败次数
	DurationSec int  `json:"duration_sec"` // 封禁时长（秒）
}

type AuthConfig struct {
	SiteTitle          string       `json:"site_title"`
	InternalNets       []string     `json:"internal_nets"`
	AdminPassword      string       `json:"-"`
	AuthPassword       string       `json:"-"`
	BarkURL            string       `json:"bark_url"`
	AdminPasswordLogin bool         `json:"admin_password_login"`
	AdminBarkLogin     bool         `json:"admin_bark_login"`
	AuthPasswordLogin  bool         `json:"auth_password_login"`
	AuthBarkLogin      bool         `json:"auth_bark_login"`
	TokenGracePeriod   int          `json:"token_grace_period"` // 宽限期（秒），0=关闭
	DefaultPolicy      Policy       `json:"default_policy"`
	Domains            []DomainRule `json:"domains"`
	AuthBan            BanConfig    `json:"auth_ban"`
	AdminBan           BanConfig    `json:"admin_ban"`
}

type ServerConfig struct {
	Listen string `json:"listen"`
}

type Config struct {
	Server ServerConfig `json:"server"`
	Auth   AuthConfig   `json:"auth"`
}

// --- 运行时状态 ---

type AppState struct {
	Config     *Config
	ParsedNets []*net.IPNet
	Sessions   *SessionStore
	Attempts   *AttemptTracker
}

// --- 常量 ---

const (
	DefaultAddr        = ":3000"
	DefaultPasswd      = "123456"
	TokenExpirySeconds = 86400
)

// --- 辅助函数 ---

func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash password:", err)
	}
	return string(hash)
}

func ParseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Printf("warning: invalid CIDR %q: %v", cidr, err)
			continue
		}
		nets = append(nets, ipNet)
	}
	return nets
}

func IsInternalIP(ipStr string, nets []*net.IPNet) bool {
	// 去掉端口
	host, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = host
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func IsValidAction(action string) bool {
	return action == "reject" || action == "pass" || action == "auth"
}

func ResolvePolicy(host string, cfg *Config) Policy {
	h := stripPort(host)
	// 第一轮：精确匹配（优先级最高）
	for _, d := range cfg.Auth.Domains {
		if d.Host == h {
			return d.Policy
		}
	}
	// 第二轮：通配符匹配（*.example.com → app.example.com）
	for _, d := range cfg.Auth.Domains {
		if matchWildcard(d.Host, h) {
			return d.Policy
		}
	}
	return cfg.Auth.DefaultPolicy
}

// stripPort 去掉 host 中的端口号
func stripPort(host string) string {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}

// matchWildcard 判断 pattern（如 *.example.com）是否匹配 host
// 规则：*.example.com 匹配 sub.example.com，不匹配 example.com
func matchWildcard(pattern, host string) bool {
	if len(pattern) < 2 || pattern[0] != '*' {
		return false
	}
	suffix := pattern[1:] // ".example.com"
	if len(host) <= len(suffix) {
		return false // host 太短，不可能匹配
	}
	return host[len(host)-len(suffix):] == suffix
}

// --- 从数据库加载配置 ---

func LoadConfig() *Config {
	cfg := &Config{}

	// 服务配置
	cfg.Server.Listen = DefaultAddr

	// 站点标题
	cfg.Auth.SiteTitle = model.ConfigGet("site_title")
	if cfg.Auth.SiteTitle == "" {
		cfg.Auth.SiteTitle = "AKGuard"
	}

	// 密码
	cfg.Auth.AdminPassword = model.ConfigGet("admin_password")
	if cfg.Auth.AdminPassword == "" {
		cfg.Auth.AdminPassword = HashPassword(DefaultPasswd)
		model.ConfigSet("admin_password", cfg.Auth.AdminPassword)
	}
	cfg.Auth.AuthPassword = model.ConfigGet("auth_password")
	if cfg.Auth.AuthPassword == "" {
		cfg.Auth.AuthPassword = HashPassword(DefaultPasswd)
		model.ConfigSet("auth_password", cfg.Auth.AuthPassword)
	}

	// Bark
	cfg.Auth.BarkURL = model.ConfigGet("bark_url")

	// 登录方式开关
	cfg.Auth.AdminPasswordLogin = model.ConfigGetBool("admin_password_login", true)
	cfg.Auth.AdminBarkLogin = model.ConfigGetBool("admin_bark_login", false)
	cfg.Auth.AuthPasswordLogin = model.ConfigGetBool("auth_password_login", true)
	cfg.Auth.AuthBarkLogin = model.ConfigGetBool("auth_bark_login", false)

	// Token 宽限期
	cfg.Auth.TokenGracePeriod = model.ConfigGetInt("token_grace_period", 0)

	// 自动封禁配置
	cfg.Auth.AuthBan = BanConfig{
		Enabled:     model.ConfigGetBool("auth_ban_enabled", false),
		WindowSec:   model.ConfigGetInt("auth_ban_window", 300),
		MaxAttempts: model.ConfigGetInt("auth_ban_max_attempts", 5),
		DurationSec: model.ConfigGetInt("auth_ban_duration", 3600),
	}
	cfg.Auth.AdminBan = BanConfig{
		Enabled:     model.ConfigGetBool("admin_ban_enabled", false),
		WindowSec:   model.ConfigGetInt("admin_ban_window", 300),
		MaxAttempts: model.ConfigGetInt("admin_ban_max_attempts", 5),
		DurationSec: model.ConfigGetInt("admin_ban_duration", 3600),
	}

	// 默认策略
	cfg.Auth.DefaultPolicy.Internal = model.ConfigGet("default_internal_policy")
	if cfg.Auth.DefaultPolicy.Internal == "" {
		cfg.Auth.DefaultPolicy.Internal = "auth"
	}
	cfg.Auth.DefaultPolicy.External = model.ConfigGet("default_external_policy")
	if cfg.Auth.DefaultPolicy.External == "" {
		cfg.Auth.DefaultPolicy.External = "auth"
	}

	// 内网网段
	nets := model.GetInternalNets()
	if nets == nil {
		nets = []string{"192.168.1.0/24", "10.0.0.0/8"}
		model.SetInternalNets(nets)
	}
	cfg.Auth.InternalNets = nets

	// 域名策略
	rules := model.GetDomainRules()
	cfg.Auth.Domains = make([]DomainRule, 0, len(rules))
	for _, r := range rules {
		cfg.Auth.Domains = append(cfg.Auth.Domains, DomainRule{
			Host:   r.Host,
			Policy: Policy{Internal: r.Internal, External: r.External},
		})
	}

	return cfg
}
