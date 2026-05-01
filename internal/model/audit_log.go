package model

import (
	"encoding/json"
)

type AuditLog struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	ClientIP  string `json:"client_ip"`
	CreatedAt string `json:"created_at"`
}

func InsertAuditLog(action string, detail interface{}, clientIP string) error {
	var detailStr string
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			detailStr = ""
		} else {
			detailStr = string(b)
		}
	}
	_, err := DB.Exec(
		`INSERT INTO audit_logs (action, detail, client_ip) VALUES (?, ?, ?)`,
		action, detailStr, clientIP,
	)
	return err
}

type PaginatedAuditLogs struct {
	Logs  []AuditLog `json:"logs"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

func QueryAuditLogs(page, pageSize int) (*PaginatedAuditLogs, error) {
	result := &PaginatedAuditLogs{Page: page, Size: pageSize}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	err := DB.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&result.Total)
	if err != nil {
		return nil, err
	}

	rows, err := DB.Query(
		"SELECT id, action, detail, client_ip, created_at FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?",
		pageSize, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result.Logs = make([]AuditLog, 0)
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(&log.ID, &log.Action, &log.Detail, &log.ClientIP, &log.CreatedAt); err != nil {
			return nil, err
		}
		result.Logs = append(result.Logs, log)
	}
	return result, nil
}
