package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all application configuration
type Config struct {
	Port         int
	RootDir      string
	DBPath       string
	SecretKey    string
	AdminUser    string
	AdminPass    string
	TrashDir     string
	MaxUpload    int64 // Max upload size in MB
	RateLimit    int   // Requests per minute
	SecureCookie bool  // Set Secure flag on cookies (for HTTPS)
	Debug        bool
}

var AppConfig *Config

// Parse initializes configuration from CLI flags
func Parse() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 8080, "Server port")
	flag.StringVar(&cfg.RootDir, "root", "/", "Root directory for file management")
	flag.StringVar(&cfg.DBPath, "db", "", "Database file path (default: ~/.filemanager/data.db)")
	flag.StringVar(&cfg.SecretKey, "secret", "", "Secret key for session encryption (auto-generated if empty)")
	flag.StringVar(&cfg.AdminUser, "admin-user", "", "Initial admin username (first run only)")
	flag.StringVar(&cfg.AdminPass, "admin-pass", "", "Initial admin password (first run only)")
	flag.StringVar(&cfg.TrashDir, "trash-dir", "", "Trash directory (default: ~/.filemanager_trash)")
	flag.Int64Var(&cfg.MaxUpload, "max-upload", 1024, "Maximum upload size in MB")
	flag.IntVar(&cfg.RateLimit, "rate-limit", 60, "Rate limit: requests per minute per IP")
	flag.BoolVar(&cfg.SecureCookie, "secure-cookie", false, "Set Secure flag on cookies (enable for HTTPS)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode")

	flag.Parse()

	// Resolve root directory
	if abs, err := filepath.Abs(cfg.RootDir); err == nil {
		cfg.RootDir = abs
	}

	// Set defaults
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}

	if cfg.DBPath == "" {
		dataDir := filepath.Join(homeDir, ".filemanager")
		os.MkdirAll(dataDir, 0750)
		cfg.DBPath = filepath.Join(dataDir, "data.db")
	}

	if cfg.TrashDir == "" {
		cfg.TrashDir = filepath.Join(homeDir, ".filemanager_trash")
	}

	if cfg.SecretKey == "" {
		cfg.SecretKey = "go-filemanager-secret-change-me-in-production"
	}

	AppConfig = cfg
	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	info, err := os.Stat(c.RootDir)
	if err != nil {
		return fmt.Errorf("root directory error: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", c.RootDir)
	}

	return nil
}
