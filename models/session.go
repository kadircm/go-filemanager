package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"go-file-manager/database"
)

// Session represents a user session
type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	CSRFToken string    `json:"csrf_token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

const sessionDuration = 24 * time.Hour * 7 // 7 days

// CreateSession creates a new session for a user
func CreateSession(userID int64) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	expiresAt := time.Now().Add(sessionDuration)

	result, err := database.DB.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	id, _ := result.LastInsertId()
	return &Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateSession checks if a session token is valid and returns the session
func ValidateSession(token string) (*Session, error) {
	session := &Session{}
	var csrfToken sql.NullString
	err := database.DB.QueryRow(
		"SELECT id, user_id, token, COALESCE(csrf_token, '') as csrf_token, expires_at, created_at FROM sessions WHERE token = ? AND expires_at > ?",
		token, time.Now(),
	).Scan(&session.ID, &session.UserID, &session.Token, &csrfToken, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}
	if csrfToken.Valid {
		session.CSRFToken = csrfToken.String
	}
	return session, nil
}

// UpdateSessionCSRF updates the CSRF token for a session
func UpdateSessionCSRF(sessionToken, csrfToken string) error {
	_, err := database.DB.Exec("UPDATE sessions SET csrf_token = ? WHERE token = ?", csrfToken, sessionToken)
	return err
}

// DeleteSession removes a session by token
func DeleteSession(token string) error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteUserSessions removes all sessions for a user
func DeleteUserSessions(userID int64) error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// CleanupExpiredSessions removes all expired sessions
func CleanupExpiredSessions() error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// generateToken creates a cryptographically secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
