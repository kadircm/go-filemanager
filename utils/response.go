package utils

import "github.com/gofiber/fiber/v2"

// APIResponse is the standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SendSuccess sends a success response
func SendSuccess(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SendError sends an error response
func SendError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(APIResponse{
		Success: false,
		Error:   message,
	})
}

// SendData sends a data-only success response
func SendData(c *fiber.Ctx, data interface{}) error {
	return c.JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}
