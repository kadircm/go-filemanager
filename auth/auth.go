package auth

import (
	"log"

	"go-file-manager/config"
	"go-file-manager/models"
	"go-file-manager/utils"

	"github.com/gofiber/fiber/v2"
)

// LoginRequest represents the login form data
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// ShowLoginPage renders the login page
func ShowLoginPage(c *fiber.Ctx) error {
	// If already logged in, redirect to files
	token := c.Cookies("session_token")
	if token != "" {
		session, _ := models.ValidateSession(token)
		if session != nil {
			return c.Redirect("/files/")
		}
	}
	return c.Render("login", fiber.Map{
		"Error": c.Query("error"),
	})
}

// HandleLogin processes login requests
func HandleLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Redirect("/login?error=Geçersiz istek")
	}

	if req.Username == "" || req.Password == "" {
		return c.Redirect("/login?error=Kullanıcı adı ve şifre gerekli")
	}

	// Find user
	user, err := models.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("Login error: %v", err)
		return c.Redirect("/login?error=Sunucu hatası")
	}
	if user == nil || !user.CheckPassword(req.Password) {
		return c.Redirect("/login?error=Geçersiz kullanıcı adı veya şifre")
	}

	// Create session
	session, err := models.CreateSession(user.ID)
	if err != nil {
		log.Printf("Session creation error: %v", err)
		return c.Redirect("/login?error=Oturum oluşturulamadı")
	}

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HTTPOnly: true,
		Secure:   config.AppConfig.SecureCookie,
		SameSite: "Lax",
	})

	return c.Redirect("/files/")
}

// HandleLogout processes logout requests
func HandleLogout(c *fiber.Ctx) error {
	token := c.Cookies("session_token")
	if token != "" {
		models.DeleteSession(token)
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HTTPOnly: true,
	})

	return c.Redirect("/login")
}

// HandleLoginAPI processes API login requests (JSON)
func HandleLoginAPI(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Geçersiz istek")
	}

	if req.Username == "" || req.Password == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Kullanıcı adı ve şifre gerekli")
	}

	user, err := models.GetUserByUsername(req.Username)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Sunucu hatası")
	}
	if user == nil || !user.CheckPassword(req.Password) {
		return utils.SendError(c, fiber.StatusUnauthorized, "Geçersiz kullanıcı adı veya şifre")
	}

	session, err := models.CreateSession(user.ID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Oturum oluşturulamadı")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HTTPOnly: true,
		Secure:   config.AppConfig.SecureCookie,
		SameSite: "Lax",
	})

	return utils.SendSuccess(c, "Giriş başarılı", fiber.Map{
		"user": user,
	})
}

// EnsureAdminExists creates the initial admin user if no users exist
func EnsureAdminExists(cfg *config.Config) {
	count, err := models.UserCount()
	if err != nil {
		log.Printf("Warning: Could not check user count: %v", err)
		return
	}

	if count > 0 {
		return // Users already exist
	}

	// Create admin from CLI flags
	username := cfg.AdminUser
	password := cfg.AdminPass

	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin123" // Default password
		log.Println("⚠ WARNING: Using default admin password 'admin123'. Change it immediately!")
	}

	user, err := models.CreateUser(username, password, "admin", cfg.RootDir)
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("✓ Admin user created: %s (role: admin, root: %s)", user.Username, user.RootDir)
}
