package services

import (
	"go-file-manager/models"
)

// CanAccessPath checks if a user can access a given path
func CanAccessPath(user *models.User, rootDir, requestPath string) bool {
	// Admin can access everything within root
	if user.IsAdmin() {
		return true
	}

	// For regular users, check if path is within their assigned root
	// The user's root_dir is relative to the server root
	// For now, users can access anything within the configured root
	return true
}

// GetUserRootDir returns the effective root directory for a user
func GetUserRootDir(user *models.User, globalRoot string) string {
	if user.RootDir != "" && user.RootDir != "/" {
		return user.RootDir
	}
	return globalRoot
}
