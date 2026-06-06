package handlers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// searchSemaphore limits concurrent search operations to prevent DoS
var searchSemaphore = make(chan struct{}, 3)

// SearchResult represents a search result
type SearchResult struct {
	Name       string             `json:"name"`
	Path       string             `json:"path"`
	IsDir      bool               `json:"is_dir"`
	Size       int64              `json:"size"`
	SizeHuman  string             `json:"size_human"`
	ModTime    time.Time          `json:"mod_time"`
	ModTimeStr string             `json:"mod_time_str"`
	Category   utils.FileCategory `json:"category"`
	ParentDir  string             `json:"parent_dir"`
}

// SearchPage renders the search results page
func SearchPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return c.Redirect("/login")
	}

	return c.Render("search", fiber.Map{
		"User":       user,
		"Query":      c.Query("q"),
		"CSRFToken":  c.Locals("csrf_token"),
		"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
	})
}

// APISearch handles search requests
func APISearch(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	query := strings.ToLower(c.Query("q"))
	if query == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Arama sorgusu gerekli")
	}

	fileType := c.Query("type")            // image, video, audio, document, code, archive
	minSize := c.QueryInt("min_size", 0)   // bytes
	maxSize := c.QueryInt("max_size", 0)
	searchTrash := c.Query("trash") == "true"

	// Acquire semaphore (limit concurrent searches)
	select {
	case searchSemaphore <- struct{}{}:
		defer func() { <-searchSemaphore }()
	default:
		return utils.SendError(c, fiber.StatusTooManyRequests, "Çok fazla eşzamanlı arama işlemi. Lütfen bekleyin.")
	}

	var results []SearchResult
	maxResults := 100
	maxDepth := 10 // Limit search depth

	// Create a context with timeout to prevent long-running searches
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu sync.Mutex

	// Search in filesystem with depth limit and timeout
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		// Check if context is cancelled (timeout)
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		if err != nil {
			return filepath.SkipDir
		}

		// Check max results
		mu.Lock()
		count := len(results)
		mu.Unlock()
		if count >= maxResults {
			return filepath.SkipAll
		}

		// Check depth limit
		relPath, _ := filepath.Rel(rootDir, path)
		depth := strings.Count(filepath.ToSlash(relPath), "/")
		if depth > maxDepth {
			return filepath.SkipDir
		}

		name := strings.ToLower(info.Name())
		if !strings.Contains(name, query) {
			return nil
		}

		// Apply filters
		category := utils.GetFileCategory(info.Name())
		if info.IsDir() {
			category = utils.CategoryFolder
		}

		if fileType != "" && string(category) != fileType && fileType != "folder" {
			return nil
		}

		if minSize > 0 && info.Size() < int64(minSize) {
			return nil
		}
		if maxSize > 0 && info.Size() > int64(maxSize) {
			return nil
		}

		relPathStr := "/" + filepath.ToSlash(relPath)
		parentDir := filepath.ToSlash(filepath.Dir(relPathStr))

		mu.Lock()
		results = append(results, SearchResult{
			Name:       info.Name(),
			Path:       relPathStr,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			SizeHuman:  utils.FormatFileSize(info.Size()),
			ModTime:    info.ModTime(),
			ModTimeStr: info.ModTime().Format("2006-01-02 15:04:05"),
			Category:   category,
			ParentDir:  parentDir,
		})
		mu.Unlock()

		return nil
	})

	// Search in trash if requested
	if searchTrash {
		trashItems, _ := services.ListTrash(config.AppConfig.TrashDir, user.Username)
		for _, item := range trashItems {
			if strings.Contains(strings.ToLower(item.Name), query) {
				results = append(results, SearchResult{
					Name:       item.Name + " (Çöp Kutusu)",
					Path:       item.OriginalPath,
					IsDir:      item.IsDir,
					Size:       item.Size,
					SizeHuman:  item.SizeHuman,
					ModTime:    item.DeletedAt,
					ModTimeStr: item.DeletedAtStr,
					Category:   utils.CategoryOther,
					ParentDir:  "Çöp Kutusu",
				})
			}
		}
	}

	return utils.SendData(c, fiber.Map{
		"query":   c.Query("q"),
		"results": results,
		"count":   len(results),
	})
}
