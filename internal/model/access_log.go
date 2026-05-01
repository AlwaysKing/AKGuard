package model

import (
	"fmt"
)

type AccessLog struct {
	ID        int64  `json:"id"`
	Host      string `json:"host"`
	ClientIP  string `json:"client_ip"`
	Source    string `json:"source"`
	Action    string `json:"action"`
	Result    string `json:"result"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
}

func InsertAccessLog(log *AccessLog) error {
	_, err := DB.Exec(
		`INSERT INTO access_logs (host, client_ip, source, action, result, user_agent) VALUES (?, ?, ?, ?, ?, ?)`,
		log.Host, log.ClientIP, log.Source, log.Action, log.Result, log.UserAgent,
	)
	return err
}

type AccessLogStats struct {
	TotalToday  int64 `json:"total_today"`
	PassCount   int64 `json:"pass_count"`
	RejectCount int64 `json:"reject_count"`
	AuthCount   int64 `json:"auth_count"`
}

func GetAccessLogStats() (*AccessLogStats, error) {
	stats := &AccessLogStats{}
	err := DB.QueryRow(
		`SELECT
			COUNT(*) as total_today,
			SUM(CASE WHEN result = 'allowed' THEN 1 ELSE 0 END) as pass_count,
			SUM(CASE WHEN result = 'denied' THEN 1 ELSE 0 END) as reject_count,
			SUM(CASE WHEN action = 'auth' THEN 1 ELSE 0 END) as auth_count
		FROM access_logs WHERE date(created_at) = date('now')`,
	).Scan(&stats.TotalToday, &stats.PassCount, &stats.RejectCount, &stats.AuthCount)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

type PaginatedAccessLogs struct {
	Logs  []AccessLog `json:"logs"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

func QueryAccessLogs(page, pageSize int, host string) (*PaginatedAccessLogs, error) {
	result := &PaginatedAccessLogs{Page: page, Size: pageSize}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 构建 WHERE 条件
	where := "WHERE 1=1"
	args := []interface{}{}
	if host != "" {
		where += " AND host LIKE ?"
		args = append(args, "%"+host+"%")
	}

	// 总数
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM access_logs %s", where)
	err := DB.QueryRow(countSQL, args...).Scan(&result.Total)
	if err != nil {
		return nil, err
	}

	// 数据
	querySQL := fmt.Sprintf(
		"SELECT id, host, client_ip, source, action, result, user_agent, created_at FROM access_logs %s ORDER BY id DESC LIMIT ? OFFSET ?",
		where,
	)
	queryArgs := append(args, pageSize, offset)
	rows, err := DB.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result.Logs = make([]AccessLog, 0)
	for rows.Next() {
		var log AccessLog
		if err := rows.Scan(&log.ID, &log.Host, &log.ClientIP, &log.Source, &log.Action, &log.Result, &log.UserAgent, &log.CreatedAt); err != nil {
			return nil, err
		}
		result.Logs = append(result.Logs, log)
	}
	return result, nil
}
