package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.RWMutex
)

// RateLimiter creates a rate limiting middleware
func RateLimiter(maxRequests int, window time.Duration) fiber.Handler {
	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > window {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		ip := c.IP()

		mu.Lock()
		v, exists := visitors[ip]
		if !exists {
			visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			mu.Unlock()
			return c.Next()
		}

		if time.Since(v.lastSeen) > window {
			v.count = 1
			v.lastSeen = time.Now()
			mu.Unlock()
			return c.Next()
		}

		v.count++
		v.lastSeen = time.Now()

		if v.count > maxRequests {
			mu.Unlock()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Çok fazla istek. Lütfen biraz bekleyin.",
			})
		}

		mu.Unlock()
		return c.Next()
	}
}

// StrictRateLimiter is a stricter rate limiter for login endpoints
func StrictRateLimiter() fiber.Handler {
	return RateLimiter(10, time.Minute) // 10 login attempts per minute
}
