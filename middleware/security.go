package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		return c.Next()
	}
}

// PathTraversalGuard prevents directory traversal attacks
func PathTraversalGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Check for path traversal patterns
		dangerousPatterns := []string{
			"..",
			"..%2f",
			"..%5c",
			"%2e%2e",
			"..\\",
		}

		lowerPath := strings.ToLower(path)
		for _, pattern := range dangerousPatterns {
			if strings.Contains(lowerPath, pattern) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"error":   "Erişim engellendi: geçersiz yol",
				})
			}
		}

		// Check for null bytes
		if strings.Contains(path, "\x00") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Erişim engellendi: geçersiz karakter",
			})
		}

		return c.Next()
	}
}
