package model

import (
	"database/sql"
	"time"
)

type BlacklistEntry struct {
	ID        int64      `json:"id"`
	IP        string     `json:"ip"`
	Type      string     `json:"type"`   // "auth" or "admin"
	Reason    string     `json:"reason"` // "manual" or "auto"
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsBlacklisted 检查 IP 是否在黑名单中（未过期）
func IsBlacklisted(ip, banType string) bool {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM blacklist WHERE ip = ? AND type = ? AND (expires_at IS NULL OR expires_at > ?)`,
		ip, banType, now,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// AddToBlacklist 添加 IP 到黑名单
func AddToBlacklist(ip, banType, reason string, duration time.Duration) error {
	var expiresAt interface{}
	if duration > 0 {
		t := time.Now().Add(duration).UTC()
		expiresAt = t.Format("2006-01-02 15:04:05")
	}
	_, err := DB.Exec(
		`INSERT INTO blacklist (ip, type, reason, expires_at) VALUES (?, ?, ?, ?)`,
		ip, banType, reason, expiresAt,
	)
	return err
}

// RemoveFromBlacklist 从黑名单移除
func RemoveFromBlacklist(id int64) error {
	_, err := DB.Exec(`DELETE FROM blacklist WHERE id = ?`, id)
	return err
}

// GetBlacklist 获取指定类型的黑名单列表
func GetBlacklist(banType string) ([]BlacklistEntry, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	rows, err := DB.Query(
		`SELECT id, ip, type, reason, expires_at, created_at FROM blacklist
		 WHERE type = ? AND (expires_at IS NULL OR expires_at > ?)
		 ORDER BY created_at DESC`, banType, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []BlacklistEntry
	for rows.Next() {
		var e BlacklistEntry
		var expiresAt sql.NullString
		var createdAt string
		if err := rows.Scan(&e.ID, &e.IP, &e.Type, &e.Reason, &expiresAt, &createdAt); err != nil {
			continue
		}
		if expiresAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", expiresAt.String)
			e.ExpiresAt = &t
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		entries = append(entries, e)
	}
	return entries, nil
}

// CleanupExpiredBlacklist 清理过期的黑名单条目
func CleanupExpiredBlacklist() {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	DB.Exec(`DELETE FROM blacklist WHERE expires_at IS NOT NULL AND expires_at <= ?`, now)
}
