package handlers

import (
	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/models"
	"go-file-manager/services"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// AdminPage renders the admin panel
func AdminPage(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.Redirect("/files/")
	}

	users, err := models.ListUsers()
	if err != nil {
		users = []*models.User{}
	}

	auditLogs, total, _ := services.GetAuditLogs(50, 0)

	return c.Render("admin", fiber.Map{
		"User":       user,
		"Users":      users,
		"AuditLogs":  auditLogs,
		"AuditTotal": total,
		"CSRFToken":  c.Locals("csrf_token"),
		"TrashCount": services.TrashItemCount(config.AppConfig.TrashDir, user.Username),
	})
}

// APIListUsers returns all users (admin only)
func APIListUsers(c *fiber.Ctx) error {
	users, err := models.ListUsers()
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Kullanıcılar listelenemedi")
	}
	return utils.SendData(c, users)
}

// APICreateUser creates a new user (admin only)
func APICreateUser(c *fiber.Ctx) error {
	admin := auth.GetCurrentUser(c)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		RootDir  string `json:"root_dir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.Username == "" || req.Password == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Kullanıcı adı ve şifre gerekli")
	}

	if req.Role == "" {
		req.Role = "user"
	}
	if req.RootDir == "" {
		req.RootDir = config.AppConfig.RootDir
	}

	// Check if username exists
	existing, _ := models.GetUserByUsername(req.Username)
	if existing != nil {
		return utils.SendError(c, fiber.StatusConflict, "Bu kullanıcı adı zaten mevcut")
	}

	user, err := models.CreateUser(req.Username, req.Password, req.Role, req.RootDir)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Kullanıcı oluşturulamadı")
	}

	services.LogAudit(admin.ID, admin.Username, services.AuditUserCreate, req.Username, "Rol: "+req.Role, c.IP())
	return utils.SendSuccess(c, "Kullanıcı oluşturuldu", user)
}

// APIUpdateUser updates a user (admin only)
func APIUpdateUser(c *fiber.Ctx) error {
	admin := auth.GetCurrentUser(c)
	userID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz kullanıcı ID")
	}

	var req struct {
		Role     string `json:"role"`
		RootDir  string `json:"root_dir"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.Role != "" || req.RootDir != "" {
		if err := models.UpdateUser(int64(userID), req.Role, req.RootDir); err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Güncelleme başarısız")
		}
	}

	if req.Password != "" {
		if err := models.UpdatePassword(int64(userID), req.Password); err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Şifre güncellenemedi")
		}
	}

	services.LogAudit(admin.ID, admin.Username, services.AuditUserUpdate, "", "Kullanıcı güncellendi", c.IP())
	return utils.SendSuccess(c, "Kullanıcı güncellendi", nil)
}

// APIDeleteUser deletes a user (admin only)
func APIDeleteUser(c *fiber.Ctx) error {
	admin := auth.GetCurrentUser(c)
	userID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz kullanıcı ID")
	}

	// Prevent self-deletion
	if int64(userID) == admin.ID {
		return utils.SendError(c, fiber.StatusBadRequest, "Kendinizi silemezsiniz")
	}

	if err := models.DeleteUser(int64(userID)); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Kullanıcı silinemedi")
	}

	// Delete user sessions
	models.DeleteUserSessions(int64(userID))

	services.LogAudit(admin.ID, admin.Username, services.AuditUserDelete, "", "Kullanıcı silindi", c.IP())
	return utils.SendSuccess(c, "Kullanıcı silindi", nil)
}

// APIGetAuditLogs returns audit logs (admin only)
func APIGetAuditLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	offset := (page - 1) * limit

	logs, total, err := services.GetAuditLogs(limit, offset)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Audit log okunamadı")
	}

	return utils.SendData(c, fiber.Map{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
