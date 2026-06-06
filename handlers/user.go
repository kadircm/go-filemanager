package handlers

import (
	"go-file-manager/auth"
	"go-file-manager/models"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// APIChangePassword allows a user to change their own password
func APIChangePassword(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Eski ve yeni şifre gerekli")
	}

	if len(req.NewPassword) < 6 {
		return utils.SendError(c, fiber.StatusBadRequest, "Yeni şifre en az 6 karakter olmalı")
	}

	// Verify old password
	if !user.CheckPassword(req.OldPassword) {
		return utils.SendError(c, fiber.StatusUnauthorized, "Eski şifre hatalı")
	}

	if err := models.UpdatePassword(user.ID, req.NewPassword); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Şifre değiştirilemedi")
	}

	return utils.SendSuccess(c, "Şifre değiştirildi", nil)
}

// APIGetCurrentUser returns the current user info
func APIGetCurrentUser(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	return utils.SendData(c, user)
}
