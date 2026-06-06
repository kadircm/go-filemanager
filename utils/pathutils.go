package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizePath cleans and validates a path to prevent directory traversal
func SanitizePath(path string) string {
	// Normalize separators
	path = filepath.ToSlash(path)

	// Remove any null bytes
	path = strings.ReplaceAll(path, "\x00", "")

	// Clean the path (resolves .., ., etc.)
	path = filepath.Clean(path)

	// Convert back to forward slashes for consistency
	path = filepath.ToSlash(path)

	return path
}

// ResolvePath safely resolves a request path within a root directory
func ResolvePath(rootDir, requestPath string) (string, error) {
	// Clean the request path
	cleaned := SanitizePath(requestPath)

	// Join with root
	fullPath := filepath.Join(rootDir, cleaned)

	// Resolve to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the resolved path is within root
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve root: %w", err)
	}

	// Normalize for comparison
	absPath = filepath.ToSlash(absPath)
	absRoot = filepath.ToSlash(absRoot)

	if !strings.HasPrefix(absPath, absRoot) {
		return "", fmt.Errorf("path traversal detected: %s is outside root %s", requestPath, rootDir)
	}

	return filepath.FromSlash(absPath), nil
}

// IsPathSafe checks if a path is safe (no traversal attempts)
func IsPathSafe(path string) bool {
	// Check for common traversal patterns
	dangerous := []string{
		"..",
		"~",
		"\x00",
	}

	cleaned := filepath.ToSlash(path)
	for _, d := range dangerous {
		if strings.Contains(cleaned, d) {
			return false
		}
	}

	return true
}

// GetRelativePath returns the path relative to root
func GetRelativePath(rootDir, fullPath string) string {
	rel, err := filepath.Rel(rootDir, fullPath)
	if err != nil {
		return fullPath
	}
	return filepath.ToSlash(rel)
}
