package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-file-manager/utils"
)

// TrashItem represents an item in the trash
type TrashItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalPath string    `json:"original_path"`
	TrashPath    string    `json:"trash_path"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
	SizeHuman    string    `json:"size_human"`
	DeletedAt    time.Time `json:"deleted_at"`
	DeletedAtStr string    `json:"deleted_at_str"`
}

// TrashInfo is the metadata stored with each trashed item
type TrashInfo struct {
	OriginalPath string    `json:"original_path"`
	DeletedAt    time.Time `json:"deleted_at"`
	Size         int64     `json:"size"`
	IsDir        bool      `json:"is_dir"`
}

// MoveToTrash moves a file or directory to the trash
func MoveToTrash(trashBaseDir, rootDir, username, filePath string) error {
	fullPath, err := utils.ResolvePath(rootDir, filePath)
	if err != nil {
		return err
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Create user's trash directory
	userTrashDir := filepath.Join(trashBaseDir, username)
	if err := os.MkdirAll(userTrashDir, 0750); err != nil {
		return fmt.Errorf("failed to create trash directory: %w", err)
	}

	// Generate unique trash ID
	trashID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fullPath))

	// Create trash item directory
	trashItemDir := filepath.Join(userTrashDir, trashID)
	if err := os.MkdirAll(trashItemDir, 0750); err != nil {
		return fmt.Errorf("failed to create trash item directory: %w", err)
	}

	// Save metadata
	trashInfo := TrashInfo{
		OriginalPath: filePath,
		DeletedAt:    time.Now(),
		Size:         info.Size(),
		IsDir:        info.IsDir(),
	}

	infoData, err := json.Marshal(trashInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal trash info: %w", err)
	}

	infoPath := filepath.Join(trashItemDir, ".trashinfo")
	if err := os.WriteFile(infoPath, infoData, 0640); err != nil {
		return fmt.Errorf("failed to write trash info: %w", err)
	}

	// Move file to trash
	destPath := filepath.Join(trashItemDir, filepath.Base(fullPath))
	if err := os.Rename(fullPath, destPath); err != nil {
		// If rename fails (cross-device), try copy + delete
		if info.IsDir() {
			if err := copyDir(fullPath, destPath); err != nil {
				return fmt.Errorf("failed to move directory to trash: %w", err)
			}
		} else {
			if err := copyFile(fullPath, destPath); err != nil {
				return fmt.Errorf("failed to move file to trash: %w", err)
			}
		}
		os.RemoveAll(fullPath)
	}

	return nil
}

// ListTrash lists all items in a user's trash
func ListTrash(trashBaseDir, username string) ([]TrashItem, error) {
	userTrashDir := filepath.Join(trashBaseDir, username)

	// Create if not exists
	if err := os.MkdirAll(userTrashDir, 0750); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(userTrashDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read trash directory: %w", err)
	}

	var items []TrashItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		trashItemDir := filepath.Join(userTrashDir, entry.Name())
		infoPath := filepath.Join(trashItemDir, ".trashinfo")

		infoData, err := os.ReadFile(infoPath)
		if err != nil {
			continue // Skip corrupted entries
		}

		var trashInfo TrashInfo
		if err := json.Unmarshal(infoData, &trashInfo); err != nil {
			continue
		}

		// Get actual file name
		subEntries, err := os.ReadDir(trashItemDir)
		if err != nil {
			continue
		}

		name := ""
		var size int64
		for _, se := range subEntries {
			if se.Name() == ".trashinfo" {
				continue
			}
			name = se.Name()
			if info, err := se.Info(); err == nil {
				size = info.Size()
			}
			break
		}

		if name == "" {
			continue
		}

		items = append(items, TrashItem{
			ID:           entry.Name(),
			Name:         name,
			OriginalPath: trashInfo.OriginalPath,
			TrashPath:    trashItemDir,
			IsDir:        trashInfo.IsDir,
			Size:         size,
			SizeHuman:    utils.FormatFileSize(size),
			DeletedAt:    trashInfo.DeletedAt,
			DeletedAtStr: trashInfo.DeletedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return items, nil
}

// RestoreFromTrash restores an item from trash to its original location
func RestoreFromTrash(trashBaseDir, rootDir, username, trashID string) error {
	userTrashDir := filepath.Join(trashBaseDir, username)
	trashItemDir := filepath.Join(userTrashDir, trashID)

	// Validate path (including Windows backslash)
	if strings.Contains(trashID, "..") || strings.Contains(trashID, "/") || strings.Contains(trashID, "\\") {
		return fmt.Errorf("invalid trash ID")
	}

	// Read trash info
	infoPath := filepath.Join(trashItemDir, ".trashinfo")
	infoData, err := os.ReadFile(infoPath)
	if err != nil {
		return fmt.Errorf("trash item not found: %w", err)
	}

	var trashInfo TrashInfo
	if err := json.Unmarshal(infoData, &trashInfo); err != nil {
		return fmt.Errorf("corrupted trash info: %w", err)
	}

	// Find the actual file in trash
	entries, err := os.ReadDir(trashItemDir)
	if err != nil {
		return fmt.Errorf("failed to read trash item: %w", err)
	}

	var fileName string
	for _, entry := range entries {
		if entry.Name() != ".trashinfo" {
			fileName = entry.Name()
			break
		}
	}

	if fileName == "" {
		return fmt.Errorf("no file found in trash item")
	}

	// Resolve original path
	originalFull, err := utils.ResolvePath(rootDir, trashInfo.OriginalPath)
	if err != nil {
		return fmt.Errorf("invalid original path: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(originalFull)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Move from trash to original location
	srcPath := filepath.Join(trashItemDir, fileName)
	if err := os.Rename(srcPath, originalFull); err != nil {
		// Cross-device fallback
		srcInfo, _ := os.Stat(srcPath)
		if srcInfo != nil && srcInfo.IsDir() {
			if err := copyDir(srcPath, originalFull); err != nil {
				return fmt.Errorf("failed to restore: %w", err)
			}
		} else {
			if err := copyFile(srcPath, originalFull); err != nil {
				return fmt.Errorf("failed to restore: %w", err)
			}
		}
	}

	// Remove trash item directory
	os.RemoveAll(trashItemDir)

	return nil
}

// DeletePermanent permanently deletes an item from trash
func DeletePermanent(trashBaseDir, username, trashID string) error {
	if strings.Contains(trashID, "..") || strings.Contains(trashID, "/") || strings.Contains(trashID, "\\") {
		return fmt.Errorf("invalid trash ID")
	}

	userTrashDir := filepath.Join(trashBaseDir, username)
	trashItemDir := filepath.Join(userTrashDir, trashID)

	return os.RemoveAll(trashItemDir)
}

// EmptyTrash removes all items from a user's trash
func EmptyTrash(trashBaseDir, username string) error {
	userTrashDir := filepath.Join(trashBaseDir, username)
	if err := os.RemoveAll(userTrashDir); err != nil {
		return fmt.Errorf("failed to empty trash: %w", err)
	}
	return os.MkdirAll(userTrashDir, 0750)
}

// TrashItemCount returns the number of items in a user's trash
func TrashItemCount(trashBaseDir, username string) int {
	items, err := ListTrash(trashBaseDir, username)
	if err != nil {
		return 0
	}
	return len(items)
}
