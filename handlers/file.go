package handlers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// dangerousExtensions contains file extensions that are blocked from upload
var dangerousExtensions = map[string]bool{
	".html": true, ".htm": true, ".svg": true,
	".jsp": true, ".asp": true, ".aspx": true,
	".php": true, ".phtml": true, ".pht": true,
	".exe": true, ".bat": true, ".cmd": true,
	".com": true, ".vbs": true, ".vbe": true,
	".wsf": true, ".wsh": true, ".msi": true,
	".scr": true, ".cpl": true, ".hta": true,
}

// chunkUploads stores in-progress chunk uploads
var chunkUploads = struct {
	sync.RWMutex
	uploads map[string]*chunkUploadState
}{uploads: make(map[string]*chunkUploadState)}

type chunkUploadState struct {
	TotalChunks int
	Received    map[int]bool
	FilePath    string
	TempDir     string
}

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
		Overwrite   bool   `json:"overwrite"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	// Check if destination exists and overwrite is not set
	if !req.Overwrite && services.FileExists(rootDir, req.Destination) {
		return utils.SendError(c, fiber.StatusConflict, "Hedef konumda aynı isimde dosya/klasör mevcut")
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
		Overwrite   bool   `json:"overwrite"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	// Check if destination exists and overwrite is not set
	if !req.Overwrite && services.FileExists(rootDir, req.Destination) {
		return utils.SendError(c, fiber.StatusConflict, "Hedef konumda aynı isimde dosya/klasör mevcut")
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
	var blockedFiles []string
	for _, file := range files {
		// Check for dangerous extensions
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if dangerousExtensions[ext] {
			blockedFiles = append(blockedFiles, file.Filename)
			continue
		}

		destPath := filepath.Join(uploadPath, file.Filename)
		fullDest, err := utils.ResolvePath(rootDir, destPath)
		if err != nil {
			continue
		}

		// Ensure directory exists
		os.MkdirAll(filepath.Dir(fullDest), 0755)
		utils.MatchParentPermissions(filepath.Dir(fullDest))

		if err := c.SaveFile(file, fullDest); err != nil {
			log.Printf("Upload error for %s: %v", file.Filename, err)
			continue
		}
		utils.MatchParentPermissions(fullDest)

		services.LogAudit(user.ID, user.Username, services.AuditUpload, destPath,
			fmt.Sprintf("Boyut: %s", utils.FormatFileSize(file.Size)), c.IP())
		uploadedCount++
	}

	if len(blockedFiles) > 0 {
		return utils.SendSuccess(c, fmt.Sprintf("%d dosya yüklendi, %d dosya güvenlik nedeniyle engellendi (%s)",
			uploadedCount, len(blockedFiles), strings.Join(blockedFiles, ", ")), nil)
	}

	return utils.SendSuccess(c, fmt.Sprintf("%d dosya yüklendi", uploadedCount), nil)
}

// APIUploadChunk handles chunked file uploads for large files
func APIUploadChunk(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	uploadID := c.FormValue("upload_id")
	chunkIndex, _ := strconv.Atoi(c.FormValue("chunk_index"))
	totalChunks, _ := strconv.Atoi(c.FormValue("total_chunks"))
	fileName := c.FormValue("filename")
	uploadPath := c.FormValue("path")

	if uploadID == "" || fileName == "" || totalChunks <= 0 {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz chunk upload parametreleri")
	}

	// Check for dangerous extensions
	ext := strings.ToLower(filepath.Ext(fileName))
	if dangerousExtensions[ext] {
		return utils.SendError(c, fiber.StatusForbidden, "Bu dosya türü güvenlik nedeniyle engellenmiştir: "+ext)
	}

	if uploadPath == "" {
		uploadPath = "/"
	}

	// Get chunk file from form
	fileHeader, err := c.FormFile("chunk")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Chunk verisi bulunamadı")
	}

	// Create temp directory for this upload
	tempDir := filepath.Join(os.TempDir(), "filemanager_chunks", uploadID)
	os.MkdirAll(tempDir, 0750)

	// Save chunk
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%05d", chunkIndex))
	if err := c.SaveFile(fileHeader, chunkPath); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Chunk kaydedilemedi")
	}

	// Track upload state
	chunkUploads.Lock()
	state, exists := chunkUploads.uploads[uploadID]
	if !exists {
		state = &chunkUploadState{
			TotalChunks: totalChunks,
			Received:    make(map[int]bool),
			FilePath:    filepath.Join(uploadPath, fileName),
			TempDir:     tempDir,
		}
		chunkUploads.uploads[uploadID] = state
	}
	state.Received[chunkIndex] = true
	allReceived := len(state.Received) == state.TotalChunks
	chunkUploads.Unlock()

	// If all chunks received, merge them
	if allReceived {
		destPath := state.FilePath
		fullDest, err := utils.ResolvePath(rootDir, destPath)
		if err != nil {
			cleanupChunks(uploadID)
			return utils.SendError(c, fiber.StatusForbidden, "Geçersiz hedef yol")
		}

		// Ensure directory exists
		os.MkdirAll(filepath.Dir(fullDest), 0755)
		utils.MatchParentPermissions(filepath.Dir(fullDest))

		// Merge chunks
		outFile, err := os.Create(fullDest)
		if err != nil {
			cleanupChunks(uploadID)
			return utils.SendError(c, fiber.StatusInternalServerError, "Dosya oluşturulamadı")
		}

		var totalSize int64
		for i := 0; i < totalChunks; i++ {
			chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%05d", i))
			chunk, err := os.Open(chunkFile)
			if err != nil {
				outFile.Close()
				cleanupChunks(uploadID)
				return utils.SendError(c, fiber.StatusInternalServerError, fmt.Sprintf("Chunk %d okunamadı", i))
			}
			n, _ := io.Copy(outFile, chunk)
			totalSize += n
			chunk.Close()
		}
		outFile.Close()

		utils.MatchParentPermissions(fullDest)
		cleanupChunks(uploadID)

		services.LogAudit(user.ID, user.Username, services.AuditUpload, destPath,
			fmt.Sprintf("Chunk upload tamamlandı, Boyut: %s", utils.FormatFileSize(totalSize)), c.IP())

		return utils.SendSuccess(c, "Dosya yüklendi", fiber.Map{
			"completed": true,
			"path":      destPath,
		})
	}

	return utils.SendSuccess(c, fmt.Sprintf("Chunk %d/%d alındı", chunkIndex+1, totalChunks), fiber.Map{
		"completed":   false,
		"chunk_index": chunkIndex,
	})
}

// cleanupChunks removes temporary chunk files
func cleanupChunks(uploadID string) {
	chunkUploads.Lock()
	state, exists := chunkUploads.uploads[uploadID]
	if exists {
		os.RemoveAll(state.TempDir)
		delete(chunkUploads.uploads, uploadID)
	}
	chunkUploads.Unlock()
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

// APICheckExists checks if a file or directory exists at the given path
func APICheckExists(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	path := c.Query("path")
	if path == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Yol parametresi gerekli")
	}

	exists := services.FileExists(rootDir, path)
	isDir := false
	if exists {
		fullPath, err := utils.ResolvePath(rootDir, path)
		if err == nil {
			if info, err := os.Stat(fullPath); err == nil {
				isDir = info.IsDir()
			}
		}
	}

	return utils.SendData(c, fiber.Map{
		"exists": exists,
		"is_dir": isDir,
		"path":   path,
	})
}

// APIChangeOwner changes the owner of a file or directory
func APIChangeOwner(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if !user.IsAdmin() {
		return utils.SendError(c, fiber.StatusForbidden, "Bu işlem için yönetici yetkisi gerekli")
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path string `json:"path"`
		UID  int    `json:"uid"`
		GID  int    `json:"gid"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	fullPath, err := utils.ResolvePath(rootDir, req.Path)
	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Erişim engellendi")
	}

	if err := utils.ChangeOwner(fullPath, req.UID, req.GID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Owner değiştirilemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditCreate, req.Path,
		fmt.Sprintf("Owner değiştirildi: UID=%d GID=%d", req.UID, req.GID), c.IP())
	return utils.SendSuccess(c, "Owner değiştirildi", nil)
}

// APIChangePermissions changes the permissions of a file or directory
func APIChangePermissions(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if !user.IsAdmin() {
		return utils.SendError(c, fiber.StatusForbidden, "Bu işlem için yönetici yetkisi gerekli")
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path       string `json:"path"`
		Permission string `json:"permission"` // e.g., "0755"
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	// Parse permission string (e.g., "0755")
	mode, err := strconv.ParseUint(req.Permission, 8, 32)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz izin formatı (örn: 0755)")
	}

	fullPath, err := utils.ResolvePath(rootDir, req.Path)
	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, "Erişim engellendi")
	}

	if err := os.Chmod(fullPath, os.FileMode(mode)); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "İzinler değiştirilemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditCreate, req.Path,
		fmt.Sprintf("İzinler değiştirildi: %s", req.Permission), c.IP())
	return utils.SendSuccess(c, "İzinler değiştirildi", nil)
}

// APIBrowseFolders returns only folders for the folder browser in copy/move dialogs
func APIBrowseFolders(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	requestPath := c.Query("path")
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

	// Sort: folders first, then files
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return utils.SendData(c, fiber.Map{
		"path":  requestPath,
		"files": files,
	})
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
