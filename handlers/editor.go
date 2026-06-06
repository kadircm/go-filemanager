package handlers

import (
	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// EditorPage renders the code editor page
func EditorPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return c.Redirect("/login")
	}

	filePath := c.Params("*")
	if filePath == "" {
		return c.Redirect("/files/")
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	content, err := services.ReadFileContent(rootDir, "/"+filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).Render("editor", fiber.Map{
			"User":  user,
			"Error": "Dosya okunamadı: " + err.Error(),
			"Path":  "/" + filePath,
		})
	}

	mode := utils.GetCodeMirrorMode(filePath)

	return c.Render("editor", fiber.Map{
		"User":      user,
		"Path":      "/" + filePath,
		"Content":   content,
		"Mode":      mode,
		"FileName":  fileBaseName(filePath),
		"CSRFToken": c.Locals("csrf_token"),
		"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
	})
}

// APIReadFile returns file content for the editor
func APIReadFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	filePath := c.Params("*")
	content, err := services.ReadFileContent(rootDir, "/"+filePath)
	if err != nil {
		return utils.SendError(c, fiber.StatusNotFound, "Dosya okunamadı: "+err.Error())
	}

	mode := utils.GetCodeMirrorMode(filePath)

	return utils.SendData(c, fiber.Map{
		"content":  content,
		"mode":     mode,
		"path":     "/" + filePath,
		"filename": fileBaseName(filePath),
	})
}

// APISaveFile saves file content from the editor
func APISaveFile(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if err := services.WriteFileContent(rootDir, req.Path, req.Content); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Dosya kaydedilemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditEdit, req.Path, "Dosya düzenlendi", c.IP())
	return utils.SendSuccess(c, "Dosya kaydedildi", nil)
}

// fileBaseName returns the base name of a file path
func fileBaseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
