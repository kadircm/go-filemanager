package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	DB   *sql.DB
	once sync.Once
)

// Initialize opens the SQLite database and runs migrations
func Initialize(dbPath string) error {
	var initErr error
	once.Do(func() {
		var err error
		DB, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// Set connection pool
		DB.SetMaxOpenConns(1) // SQLite handles one writer at a time
		DB.SetMaxIdleConns(1)

		if err := DB.Ping(); err != nil {
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		if err := runMigrations(); err != nil {
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		log.Println("✓ Database initialized:", dbPath)
	})
	return initErr
}

// runMigrations creates tables if they don't exist
func runMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin', 'user')),
			root_dir TEXT NOT NULL DEFAULT '/',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_path TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at)`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s: %w", m[:50], err)
		}
	}

	return nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}
