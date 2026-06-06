package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go-file-manager/utils"
)

// FileInfo represents a file or directory entry
type FileInfo struct {
	Name        string             `json:"name"`
	Path        string             `json:"path"`
	IsDir       bool               `json:"is_dir"`
	Size        int64              `json:"size"`
	SizeHuman   string             `json:"size_human"`
	ModTime     time.Time          `json:"mod_time"`
	ModTimeStr  string             `json:"mod_time_str"`
	Permissions string             `json:"permissions"`
	Category    utils.FileCategory `json:"category"`
	MimeType    string             `json:"mime_type,omitempty"`
	Owner       string             `json:"owner,omitempty"`
	IsSymlink   bool               `json:"is_symlink"`
	Extension   string             `json:"extension"`
}

// ListDirectory lists the contents of a directory
func ListDirectory(rootDir, requestPath string) ([]FileInfo, error) {
	fullPath, err := utils.ResolvePath(rootDir, requestPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't read
		}

		relPath := filepath.Join(requestPath, entry.Name())
		relPath = filepath.ToSlash(relPath)
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}

		fi := FileInfo{
			Name:        entry.Name(),
			Path:        relPath,
			IsDir:       entry.IsDir(),
			Size:        info.Size(),
			SizeHuman:   utils.FormatFileSize(info.Size()),
			ModTime:     info.ModTime(),
			ModTimeStr:  info.ModTime().Format("2006-01-02 15:04:05"),
			Permissions: info.Mode().String(),
			IsSymlink:   info.Mode()&os.ModeSymlink != 0,
			Extension:   strings.ToLower(filepath.Ext(entry.Name())),
		}

		if entry.IsDir() {
			fi.Category = utils.CategoryFolder
		} else {
			fi.Category = utils.GetFileCategory(entry.Name())
			fi.MimeType = utils.GetMimeType(entry.Name())
		}

		files = append(files, fi)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// CreateDirectory creates a new directory
func CreateDirectory(rootDir, dirPath string) error {
	fullPath, err := utils.ResolvePath(rootDir, dirPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return err
	}
	return utils.MatchParentPermissions(fullPath)
}

// CreateFile creates a new empty file
func CreateFile(rootDir, filePath string) error {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	parent := filepath.Dir(fullPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	utils.MatchParentPermissions(parent)

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	f.Close()
	return utils.MatchParentPermissions(fullPath)
}

// RenameFile renames a file or directory
func RenameFile(rootDir, oldPath, newName string) error {
	oldFull, err := utils.ResolvePath(rootDir, oldPath)
	if err != nil {
		return err
	}

	newFull := filepath.Join(filepath.Dir(oldFull), newName)

	// Ensure new path is still within root
	newRelPath := filepath.Join(filepath.Dir(oldPath), newName)
	if _, err := utils.ResolvePath(rootDir, newRelPath); err != nil {
		return err
	}

	return os.Rename(oldFull, newFull)
}

// MoveFile moves a file or directory to a new location
func MoveFile(rootDir, srcPath, dstPath string) error {
	srcFull, err := utils.ResolvePath(rootDir, srcPath)
	if err != nil {
		return err
	}

	dstFull, err := utils.ResolvePath(rootDir, dstPath)
	if err != nil {
		return err
	}

	return os.Rename(srcFull, dstFull)
}

// CopyFile copies a file to a new location
func CopyFile(rootDir, srcPath, dstPath string) error {
	srcFull, err := utils.ResolvePath(rootDir, srcPath)
	if err != nil {
		return err
	}

	dstFull, err := utils.ResolvePath(rootDir, dstPath)
	if err != nil {
		return err
	}

	// Check if source is a directory
	srcInfo, err := os.Stat(srcFull)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	if srcInfo.IsDir() {
		return copyDir(srcFull, dstFull)
	}

	return copyFile(srcFull, dstFull)
}

// ReadFileContent reads the content of a file
func ReadFileContent(rootDir, filePath string) (string, error) {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// WriteFileContent writes content to a file
func WriteFileContent(rootDir, filePath, content string) error {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return err
	}

	// Get existing file permissions or use default
	mode := os.FileMode(0644)
	if info, err := os.Stat(fullPath); err == nil {
		mode = info.Mode()
	}

	if err := os.WriteFile(fullPath, []byte(content), mode); err != nil {
		return err
	}
	return utils.MatchParentPermissions(fullPath)
}

// GetFileInfo returns info about a file or directory
func GetFileInfo(rootDir, filePath string) (*FileInfo, error) {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	fi := &FileInfo{
		Name:        info.Name(),
		Path:        filepath.ToSlash(filePath),
		IsDir:       info.IsDir(),
		Size:        info.Size(),
		SizeHuman:   utils.FormatFileSize(info.Size()),
		ModTime:     info.ModTime(),
		ModTimeStr:  info.ModTime().Format("2006-01-02 15:04:05"),
		Permissions: info.Mode().String(),
		Extension:   strings.ToLower(filepath.Ext(info.Name())),
	}

	if info.IsDir() {
		fi.Category = utils.CategoryFolder
	} else {
		fi.Category = utils.GetFileCategory(info.Name())
		fi.MimeType = utils.GetMimeType(info.Name())
	}

	return fi, nil
}

// FileExists checks if a file or directory exists
func FileExists(rootDir, filePath string) bool {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullPath)
	return err == nil
}

// GetFullPath resolves and returns the full filesystem path
func GetFullPath(rootDir, filePath string) (string, error) {
	return utils.ResolvePath(rootDir, filePath)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	utils.MatchParentPermissions(filepath.Dir(dst))

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return utils.MatchParentPermissions(dst)
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	utils.MatchParentPermissions(dst)

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
