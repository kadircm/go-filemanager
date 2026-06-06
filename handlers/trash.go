package handlers

import (
	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// TrashPage renders the trash page
func TrashPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return c.Redirect("/login")
	}

	items, err := services.ListTrash(config.AppConfig.TrashDir, user.Username)
	if err != nil {
		items = []services.TrashItem{}
	}

	return c.Render("trash", fiber.Map{
		"User":       user,
		"Items":      items,
		"TrashCount": len(items),
		"CSRFToken":  c.Locals("csrf_token"),
	})
}

// APIListTrash returns trash items as JSON
func APIListTrash(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	items, err := services.ListTrash(config.AppConfig.TrashDir, user.Username)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Çöp kutusu okunamadı")
	}

	if items == nil {
		items = []services.TrashItem{}
	}

	return utils.SendData(c, fiber.Map{
		"items": items,
		"count": len(items),
	})
}

// APIRestoreTrash restores an item from trash
func APIRestoreTrash(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	var req struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.ID == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "ID gerekli")
	}

	rootDir := services.GetUserRootDir(user, config.AppConfig.RootDir)

	if err := services.RestoreFromTrash(config.AppConfig.TrashDir, rootDir, user.Username, req.ID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Geri yüklenemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditRestore, req.ID, "Çöp kutusundan geri yüklendi", c.IP())
	return utils.SendSuccess(c, "Geri yüklendi", nil)
}

// APIDeleteTrashItem permanently deletes a trash item
func APIDeleteTrashItem(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	trashID := c.Params("id")

	if trashID == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "ID gerekli")
	}

	if err := services.DeletePermanent(config.AppConfig.TrashDir, user.Username, trashID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Kalıcı silinemedi: "+err.Error())
	}

	services.LogAudit(user.ID, user.Username, services.AuditDelete, trashID, "Kalıcı olarak silindi", c.IP())
	return utils.SendSuccess(c, "Kalıcı olarak silindi", nil)
}

// APIEmptyTrash empties the user's trash
func APIEmptyTrash(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	if err := services.EmptyTrash(config.AppConfig.TrashDir, user.Username); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Çöp kutusu boşaltılamadı")
	}

	services.LogAudit(user.ID, user.Username, services.AuditDelete, "trash", "Çöp kutusu boşaltıldı", c.IP())
	return utils.SendSuccess(c, "Çöp kutusu boşaltıldı", nil)
}
