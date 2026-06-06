package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"go-file-manager/models"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth middleware checks for a valid session
func RequireAuth(c *fiber.Ctx) error {
	token := c.Cookies("session_token")
	if token == "" {
		// Check if API request
		if isAPIRequest(c) {
			return utils.SendError(c, fiber.StatusUnauthorized, "Oturum gerekli")
		}
		return c.Redirect("/login")
	}

	session, err := models.ValidateSession(token)
	if err != nil {
		log.Printf("Session validation error: %v", err)
		if isAPIRequest(c) {
			return utils.SendError(c, fiber.StatusInternalServerError, "Sunucu hatası")
		}
		return c.Redirect("/login")
	}

	if session == nil {
		// Session expired or invalid - clear cookie
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HTTPOnly: true,
		})
		if isAPIRequest(c) {
			return utils.SendError(c, fiber.StatusUnauthorized, "Oturum süresi dolmuş")
		}
		return c.Redirect("/login")
	}

	// Get user
	user, err := models.GetUserByID(session.UserID)
	if err != nil || user == nil {
		if isAPIRequest(c) {
			return utils.SendError(c, fiber.StatusUnauthorized, "Kullanıcı bulunamadı")
		}
		return c.Redirect("/login")
	}

	// Store user in context
	c.Locals("user", user)
	c.Locals("session", session)

	return c.Next()
}

// RequireAdmin middleware checks for admin role
func RequireAdmin(c *fiber.Ctx) error {
	user := GetCurrentUser(c)
	if user == nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Oturum gerekli")
	}

	if !user.IsAdmin() {
		return utils.SendError(c, fiber.StatusForbidden, "Yönetici yetkisi gerekli")
	}

	return c.Next()
}

// GetCurrentUser retrieves the current user from context
func GetCurrentUser(c *fiber.Ctx) *models.User {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

// CSRFMiddleware generates and validates CSRF tokens
func CSRFMiddleware(c *fiber.Ctx) error {
	// Generate CSRF token if not present
	csrfToken := c.Cookies("csrf_token")
	if csrfToken == "" {
		csrfToken = generateCSRFToken()
		c.Cookie(&fiber.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HTTPOnly: false, // JS needs to read this
			SameSite: "Lax",
		})
	}

	// For safe methods, just pass through
	method := c.Method()
	if method == "GET" || method == "HEAD" || method == "OPTIONS" {
		c.Locals("csrf_token", csrfToken)
		return c.Next()
	}

	// For state-changing methods, validate CSRF token
	requestToken := c.Get("X-CSRF-Token")
	if requestToken == "" {
		requestToken = c.FormValue("_csrf")
	}

	if requestToken == "" || requestToken != csrfToken {
		return utils.SendError(c, fiber.StatusForbidden, "Geçersiz CSRF token")
	}

	c.Locals("csrf_token", csrfToken)
	return c.Next()
}

// isAPIRequest checks if the request expects JSON
func isAPIRequest(c *fiber.Ctx) bool {
	return c.Get("Accept") == "application/json" ||
		c.Get("Content-Type") == "application/json" ||
		c.Get("X-Requested-With") == "XMLHttpRequest" ||
		len(c.Path()) > 4 && c.Path()[:5] == "/api/"
}

// generateCSRFToken creates a new CSRF token
func generateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
