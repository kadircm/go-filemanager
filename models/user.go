package models

import (
	"database/sql"
	"fmt"
	"time"

	"go-file-manager/database"

	"golang.org/x/crypto/bcrypt"
)

// User represents a system user
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" or "user"
	RootDir      string    `json:"root_dir"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateUser creates a new user with hashed password
func CreateUser(username, password, role, rootDir string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (username, password_hash, role, root_dir) VALUES (?, ?, ?, ?)",
		username, string(hash), role, rootDir,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:       id,
		Username: username,
		Role:     role,
		RootDir:  rootDir,
	}, nil
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, role, root_dir, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.RootDir, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, role, root_dir, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.RootDir, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// ListUsers returns all users
func ListUsers() ([]*User, error) {
	rows, err := database.DB.Query(
		"SELECT id, username, role, root_dir, created_at, updated_at FROM users ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.RootDir, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateUser updates a user's details
func UpdateUser(id int64, role, rootDir string) error {
	_, err := database.DB.Exec(
		"UPDATE users SET role = ?, root_dir = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		role, rootDir, id,
	)
	return err
}

// UpdatePassword updates a user's password
func UpdatePassword(id int64, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = database.DB.Exec(
		"UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(hash), id,
	)
	return err
}

// DeleteUser removes a user by ID
func DeleteUser(id int64) error {
	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// CheckPassword verifies a password against the hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// UserCount returns the number of users
func UserCount() (int, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
