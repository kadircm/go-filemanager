package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// FilesPage renders the file manager page
func FilesPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return c.Redirect("/login")
	}

	requestPath := c.Params("*")
	if requestPath == "" {
		requestPath = "/"
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	files, err := services.ListDirectory(rootDir, requestPath)
	if err != nil {
		log.Printf("Error listing directory: %v", err)
		return c.Status(fiber.StatusNotFound).Render("files", fiber.Map{
			"User":       user,
			"Error":      "Dizin bulunamadı: " + requestPath,
			"Path":       requestPath,
			"Files":      []services.FileInfo{},
			"Breadcrumb": buildBreadcrumb(requestPath),
			"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
		})
	}

	return c.Render("files", fiber.Map{
		"User":       user,
		"Path":       requestPath,
		"Files":      files,
		"Breadcrumb": buildBreadcrumb(requestPath),
		"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
		"CSRFToken":  c.Locals("csrf_token"),
	})
}

// APIListFiles returns directory listing as JSON
func APIListFiles(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	requestPath := c.Params("*")
	if requestPath == "" {
		requestPath = "/"
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	files, err := services.ListDirectory(rootDir, requestPath)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Dizin bulunamadı: "+err.Error())
	}

	return utils.SendData(c, fiber.Map{
		"path":  requestPath,
		"files": files,
	})
}

// APICreateDir creates a new directory
func APICreateDir(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.Name == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Klasör adı gerekli")
	}

	dirPath := filepath.Join(req.Path, req.Name)
	if err := services.CreateDirectory(rootDir, dirPath); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Klasör oluşturulamadı: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditCreate, dirPath, "Klasör oluşturuldu", c.IP())
	return utils.SendSuccess(c, "Klasör oluşturuldu", nil)
}

// APICreateFile creates a new empty file
func APICreateFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.Name == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Dosya adı gerekli")
	}

	filePath := filepath.Join(req.Path, req.Name)
	if err := services.CreateFile(rootDir, filePath); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Dosya oluşturulamadı: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditCreate, filePath, "Dosya oluşturuldu", c.IP())
	return utils.SendSuccess(c, "Dosya oluşturuldu", nil)
}

// APIRenameFile renames a file or directory
func APIRenameFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if err := services.RenameFile(rootDir, req.Path, req.NewName); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Yeniden adlandırılamadı: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditRename, req.Path, "Yeni ad: "+req.NewName, c.IP())
	return utils.SendSuccess(c, "Yeniden adlandırıldı", nil)
}

// APIMoveFile moves a file or directory
func APIMoveFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if err := services.MoveFile(rootDir, req.Source, req.Destination); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Taşınamadı: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditMove, req.Source, "Hedef: "+req.Destination, c.IP())
	return utils.SendSuccess(c, "Taşındı", nil)
}

// APICopyFile copies a file or directory
func APICopyFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if err := services.CopyFile(rootDir, req.Source, req.Destination); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Kopyalanamadı: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditCopy, req.Source, "Hedef: "+req.Destination, c.IP())
	return utils.SendSuccess(c, "Kopyalandı", nil)
}

// APIDeleteFile moves a file to trash
func APIDeleteFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	filePath := c.Params("*")
	if filePath == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Dosya yolu gerekli")
	}

	if err := services.MoveToTrash(config.AppConfig.TrashDir, rootDir, user.Username, "/"+filePath); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Silinemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditTrash, filePath, "Çöp kutusuna taşındı", c.IP())
	return utils.SendSuccess(c, "Çöp kutusuna taşındı", nil)
}

// APIUploadFile handles file uploads
func APIUploadFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	uploadPath := c.FormValue("path")
	if uploadPath == "" {
		uploadPath = "/"
	}

	form, err := c.MultipartForm()
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz yükleme")
	}

	files := form.File["files"]
	if len(files) == 0 {
		return utils.SendError(c, fiber.StatusBadRequest, "Dosya seçilmedi")
	}

	uploadedCount := 0
	for _, file := range files {
		destPath := filepath.Join(uploadPath, file.Filename)
		fullDest, err := utils.ResolvePath(rootDir, destPath)
		if err != nil {
			continue
		}

		// Ensure directory exists
		os.MkdirAll(filepath.Dir(fullDest), 0755)

		if err := c.SaveFile(file, fullDest); err != nil {
			log.Printf("Upload error for %s: %v", file.Filename, err)
			continue
		}

		services.LogAudit(user.ID, user.Username, services.AuditUpload, destPath,
			fmt.Sprintf("Boyut: %s", utils.FormatFileSize(file.Size)), c.IP())
		uploadedCount++
	}

	return utils.SendSuccess(c, fmt.Sprintf("%d dosya yüklendi", uploadedCount), nil)
}

// APIDownloadFile handles file downloads
func APIDownloadFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	filePath := c.Params("*")
	fullPath, err := utils.ResolvePath(rootDir, "/"+filePath)
	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Erişim engellendi")
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Dosya bulunamadı")
	}

	if info.IsDir() {
		return utils.SendError(c, fiber.StatusBadRequest, "Klasör indirilemez")
	}

	services.LogAudit(user.ID, user.Username, services.AuditDownload, filePath, "", c.IP())
	return c.Download(fullPath, filepath.Base(fullPath))
}

// APIGetFileInfo returns file information
func APIGetFileInfo(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	filePath := c.Params("*")
	info, err := services.GetFileInfo(rootDir, "/"+filePath)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Dosya bulunamadı")
	}

	return utils.SendData(c, info)
}

// BreadcrumbItem represents a breadcrumb navigation item
type BreadcrumbItem struct {
	Name string
	Path string
}

// buildBreadcrumb creates breadcrumb navigation items from a path
func buildBreadcrumb(path string) []BreadcrumbItem {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var items []BreadcrumbItem

	items = append(items, BreadcrumbItem{Name: "Ana Dizin", Path: "/"})

	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		currentPath += "/" + part
		items = append(items, BreadcrumbItem{Name: part, Path: currentPath})
	}

	return items
}
