package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// MediaPage renders the media player page
func MediaPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return c.Redirect("/login")
	}

	filePath := c.Params("*")
	if filePath == "" {
		return c.Redirect("/files/")
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)
	info, err := services.GetFileInfo(rootDir, "/"+filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).Render("media", fiber.Map{
			"User":  user,
			"Error": "Dosya bulunamadı",
		})
	}

	category := utils.GetFileCategory(filePath)

	return c.Render("media", fiber.Map{
		"User":       user,
		"Path":       "/" + filePath,
		"FileName":   info.Name,
		"FileInfo":   info,
		"Category":   string(category),
		"MimeType":   info.MimeType,
		"StreamURL":  "/api/media/stream/" + filePath,
		"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
		"CSRFToken":  c.Locals("csrf_token"),
	})
}

// APIMediaStream handles HTTP 206 Partial Content streaming
func APIMediaStream(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	filePath := c.Params("*")
	fullPath, err := utils.ResolvePath(rootDir, "/"+filePath)
	if err != nil {
		return c.Status(fiber.StatusForbidden).SendString("Access denied")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Cannot read file")
	}

	fileSize := stat.Size()
	mimeType := utils.GetMimeType(filePath)

	// Handle Range header for partial content
	rangeHeader := c.Get("Range")
	if rangeHeader != "" {
		return handleRangeRequest(c, file, fileSize, mimeType, rangeHeader)
	}

	// Full content response
	c.Set("Content-Type", mimeType)
	c.Set("Content-Length", strconv.FormatInt(fileSize, 10))
	c.Set("Accept-Ranges", "bytes")

	return c.SendFile(fullPath)
}

// handleRangeRequest handles HTTP 206 Partial Content
func handleRangeRequest(c *fiber.Ctx, file *os.File, fileSize int64, mimeType, rangeHeader string) error {
	// Parse Range header (e.g., "bytes=0-1024")
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeHeader, "-")

	var start, end int64

	if parts[0] != "" {
		s, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return c.Status(http.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
		}
		start = s
	}

	if len(parts) > 1 && parts[1] != "" {
		e, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return c.Status(http.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
		}
		end = e
	} else {
		// Default: serve 1MB chunks
		end = start + 1024*1024 - 1
		if end >= fileSize {
			end = fileSize - 1
		}
	}

	if start >= fileSize || start > end {
		c.Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		return c.Status(http.StatusRequestedRangeNotSatisfiable).SendString("Range not satisfiable")
	}

	if end >= fileSize {
		end = fileSize - 1
	}

	contentLength := end - start + 1

	c.Status(http.StatusPartialContent)
	c.Set("Content-Type", mimeType)
	c.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Set("Accept-Ranges", "bytes")

	// Seek to start position
	file.Seek(start, io.SeekStart)

	// Read and send the range
	buf := make([]byte, contentLength)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return c.Status(fiber.StatusInternalServerError).SendString("Read error")
	}

	return c.Send(buf[:n])
}
