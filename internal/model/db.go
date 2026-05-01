package model

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA busy_timeout=5000")

	if err = createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	log.Printf("database initialized: %s", dbPath)
	return nil
}

func createTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS internal_nets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cidr TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS domain_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host TEXT NOT NULL UNIQUE,
			internal_policy TEXT NOT NULL DEFAULT 'auth',
			external_policy TEXT NOT NULL DEFAULT 'auth'
		)`,
		`CREATE TABLE IF NOT EXISTS access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host TEXT NOT NULL,
			client_ip TEXT NOT NULL,
			source TEXT NOT NULL,
			action TEXT NOT NULL,
			result TEXT NOT NULL,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_host ON access_logs(host)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_created ON access_logs(created_at)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			detail TEXT,
			client_ip TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			client_ip TEXT,
			user_agent TEXT,
			token_type TEXT NOT NULL,
			is_init BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,
			`CREATE TABLE IF NOT EXISTS blacklist (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				ip TEXT NOT NULL,
				type TEXT NOT NULL,
				reason TEXT NOT NULL,
				expires_at DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_blacklist_type_ip ON blacklist(type, ip)`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// ConfigGet 读取配置值
func ConfigGet(key string) string {
	var val string
	err := DB.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

// ConfigSet 写入配置值
func ConfigSet(key, value string) {
	DB.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, value)
}

// ConfigGetBool 读取布尔配置值，默认返回 def
func ConfigGetBool(key string, def bool) bool {
	val := ConfigGet(key)
	if val == "" {
		return def
	}
	return val == "1" || val == "true"
}

// ConfigGetInt 读取整数配置值，默认返回 def
func ConfigGetInt(key string, def int) int {
	val := ConfigGet(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}

// GetInternalNets 获取内网网段列表
func GetInternalNets() []string {
	rows, err := DB.Query("SELECT cidr FROM internal_nets ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var nets []string
	for rows.Next() {
		var cidr string
		if err := rows.Scan(&cidr); err == nil {
			nets = append(nets, cidr)
		}
	}
	return nets
}

// SetInternalNets 替换内网网段列表
func SetInternalNets(nets []string) {
	tx, _ := DB.Begin()
	tx.Exec("DELETE FROM internal_nets")
	for _, cidr := range nets {
		tx.Exec("INSERT INTO internal_nets (cidr) VALUES (?)", cidr)
	}
	tx.Commit()
}

// DomainRuleDB 域名策略
type DomainRuleDB struct {
	Host     string `json:"host"`
	Internal string `json:"internal"`
	External string `json:"external"`
}

// GetDomainRules 获取域名策略列表
func GetDomainRules() []DomainRuleDB {
	rows, err := DB.Query("SELECT host, internal_policy, external_policy FROM domain_rules ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var rules []DomainRuleDB
	for rows.Next() {
		var r DomainRuleDB
		if err := rows.Scan(&r.Host, &r.Internal, &r.External); err == nil {
			rules = append(rules, r)
		}
	}
	return rules
}

// SetDomainRules 替换域名策略列表
func SetDomainRules(rules []DomainRuleDB) {
	tx, _ := DB.Begin()
	tx.Exec("DELETE FROM domain_rules")
	for _, r := range rules {
		tx.Exec("INSERT INTO domain_rules (host, internal_policy, external_policy) VALUES (?, ?, ?)",
			r.Host, r.Internal, r.External)
	}
	tx.Commit()
}
