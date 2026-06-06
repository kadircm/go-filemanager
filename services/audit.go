package services

import (
	"go-file-manager/database"
)

// AuditAction types
const (
	AuditCreate     = "create"
	AuditDelete     = "delete"
	AuditRename     = "rename"
	AuditMove       = "move"
	AuditCopy       = "copy"
	AuditUpload     = "upload"
	AuditDownload   = "download"
	AuditEdit       = "edit"
	AuditTrash      = "trash"
	AuditRestore    = "restore"
	AuditPermission = "permission"
	AuditLogin      = "login"
	AuditLogout     = "logout"
	AuditUserCreate = "user_create"
	AuditUserDelete = "user_delete"
	AuditUserUpdate = "user_update"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Action       string `json:"action"`
	ResourcePath string `json:"resource_path"`
	Details      string `json:"details"`
	IPAddress    string `json:"ip_address"`
	CreatedAt    string `json:"created_at"`
}

// LogAudit records an action in the audit log
func LogAudit(userID int64, username, action, resourcePath, details, ipAddress string) {
	_, _ = database.DB.Exec(
		"INSERT INTO audit_logs (user_id, username, action, resource_path, details, ip_address) VALUES (?, ?, ?, ?, ?, ?)",
		userID, username, action, resourcePath, details, ipAddress,
	)
}

// GetAuditLogs retrieves audit log entries with pagination
func GetAuditLogs(limit, offset int) ([]AuditEntry, int, error) {
	var total int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := database.DB.Query(
		"SELECT id, user_id, username, action, resource_path, COALESCE(details, ''), COALESCE(ip_address, ''), created_at FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.ResourcePath, &e.Details, &e.IPAddress, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}
