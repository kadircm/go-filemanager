package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-file-manager/auth"
	"go-file-manager/config"
	"go-file-manager/database"
	"go-file-manager/handlers"
	"go-file-manager/middleware"
	"go-file-manager/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
)

//go:embed web/templates/*.html
var templateFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

func main() {
	// Parse configuration
	cfg := config.Parse()

	// Validate
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database
	if err := database.Initialize(cfg.DBPath); err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer database.Close()

	// Create initial admin user if needed
	auth.EnsureAdminExists(cfg)

	// Setup template engine
	engine := setupTemplateEngine()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views:                 engine,
		BodyLimit:             int(cfg.MaxUpload) * 1024 * 1024,
		DisableStartupMessage: false,
		ErrorHandler:          customErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(middleware.Logger())
	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.PathTraversalGuard())

	// Serve static files from embedded FS
	staticSub, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatalf("Failed to setup static files: %v", err)
	}
	app.Use("/static", filesystem(staticSub))

	// ========================================
	// Public Routes
	// ========================================
	app.Get("/login", auth.ShowLoginPage)
	app.Post("/login", middleware.StrictRateLimiter(), auth.HandleLogin)
	app.Post("/api/login", middleware.StrictRateLimiter(), auth.HandleLoginAPI)
	app.Get("/logout", auth.HandleLogout)

	// Redirect root to files
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/files/")
	})

	// ========================================
	// Authenticated Routes
	// ========================================
	authenticated := app.Group("", auth.RequireAuth, auth.CSRFMiddleware)

	// Rate limit for general API
	apiLimited := authenticated.Group("", middleware.RateLimiter(cfg.RateLimit, time.Minute))

	// Page routes
	authenticated.Get("/files/*", handlers.FilesPage)
	authenticated.Get("/trash/", handlers.TrashPage)
	authenticated.Get("/editor/*", func(c *fiber.Ctx) error {
		// Set EditorMode for layout to load CodeMirror
		c.Locals("EditorMode", true)
		return handlers.EditorPage(c)
	})
	authenticated.Get("/media/*", handlers.MediaPage)
	authenticated.Get("/search/", handlers.SearchPage)

	// File API routes
	apiLimited.Get("/api/files/list/*", handlers.APIListFiles)
	apiLimited.Post("/api/files/mkdir", handlers.APICreateDir)
	apiLimited.Post("/api/files/create", handlers.APICreateFile)
	apiLimited.Put("/api/files/rename", handlers.APIRenameFile)
	apiLimited.Put("/api/files/move", handlers.APIMoveFile)
	apiLimited.Post("/api/files/copy", handlers.APICopyFile)
	apiLimited.Delete("/api/files/*", handlers.APIDeleteFile)
	apiLimited.Post("/api/files/upload", handlers.APIUploadFile)
	apiLimited.Post("/api/files/upload/chunk", handlers.APIUploadChunk)
	apiLimited.Get("/api/files/download/*", handlers.APIDownloadFile)
	apiLimited.Get("/api/files/info/*", handlers.APIGetFileInfo)
	apiLimited.Get("/api/files/check-exists", handlers.APICheckExists)
	apiLimited.Get("/api/files/browse", handlers.APIBrowseFolders)
	apiLimited.Put("/api/files/chown", handlers.APIChangeOwner)
	apiLimited.Put("/api/files/chmod", handlers.APIChangePermissions)

	// Editor API routes
	apiLimited.Get("/api/editor/read/*", handlers.APIReadFile)
	apiLimited.Post("/api/editor/save", handlers.APISaveFile)

	// Trash API routes
	apiLimited.Get("/api/trash", handlers.APIListTrash)
	apiLimited.Post("/api/trash/restore", handlers.APIRestoreTrash)
	apiLimited.Delete("/api/trash/:id", handlers.APIDeleteTrashItem)
	apiLimited.Delete("/api/trash", handlers.APIEmptyTrash)

	// Media API routes
	apiLimited.Get("/api/media/stream/*", handlers.APIMediaStream)

	// Search API routes
	apiLimited.Get("/api/search", handlers.APISearch)

	// User API routes
	apiLimited.Get("/api/user/me", handlers.APIGetCurrentUser)
	apiLimited.Post("/api/user/password", handlers.APIChangePassword)

	// Admin routes
	admin := authenticated.Group("", auth.RequireAdmin)
	admin.Get("/admin/", handlers.AdminPage)
	admin.Get("/api/admin/users", handlers.APIListUsers)
	admin.Post("/api/admin/users", handlers.APICreateUser)
	admin.Put("/api/admin/users/:id", handlers.APIUpdateUser)
	admin.Delete("/api/admin/users/:id", handlers.APIDeleteUser)
	admin.Get("/api/admin/audit", handlers.APIGetAuditLogs)

	// ========================================
	// Session Cleanup (background)
	// ========================================
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			if err := models.CleanupExpiredSessions(); err != nil {
				log.Printf("Session cleanup error: %v", err)
			}
		}
	}()

	// ========================================
	// Start Server
	// ========================================
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("═══════════════════════════════════════")
	log.Printf("  Go File Manager")
	log.Printf("  Port: %d", cfg.Port)
	log.Printf("  Root: %s", cfg.RootDir)
	log.Printf("  DB:   %s", cfg.DBPath)
	log.Printf("  URL:  http://localhost%s", addr)
	log.Printf("═══════════════════════════════════════")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("\n⏹ Shutting down...")
		app.Shutdown()
	}()

	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// setupTemplateEngine creates the HTML template engine with custom functions
func setupTemplateEngine() *html.Engine {
	// Use fs.Sub to get the web/templates subdirectory
	templateSub, err := fs.Sub(templateFS, "web/templates")
	if err != nil {
		log.Fatalf("Failed to setup template FS: %v", err)
	}

	engine := html.NewFileSystem(http.FS(templateSub), ".html")

	engine.AddFunc("sub", func(a, b int) int {
		return a - b
	})

	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})

	engine.AddFunc("len", func(v interface{}) int {
		switch val := v.(type) {
		case string:
			return len(val)
		case []interface{}:
			return len(val)
		default:
			return 0
		}
	})

	engine.AddFunc("slice", func(s string, start, end int) string {
		if start >= len(s) {
			return ""
		}
		if end > len(s) {
			end = len(s)
		}
		return strings.ToUpper(s[start:end])
	})

	engine.AddFunc("safe", func(s string) template.HTML {
		return template.HTML(s)
	})

	engine.AddFunc("eq", func(a, b interface{}) bool {
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	})

	engine.Reload(true) // Enable reload during development

	return engine
}

// filesystem creates a handler for serving embedded static files
func filesystem(fsys fs.FS) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		// Strip the /static/ prefix
		path = strings.TrimPrefix(path, "/static/")
		if path == "" {
			path = "index.html"
		}

		file, err := fsys.Open(path)
		if err != nil {
			return c.Next()
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil || stat.IsDir() {
			return c.Next()
		}

		// Set content type
		contentType := "application/octet-stream"
		if strings.HasSuffix(path, ".css") {
			contentType = "text/css; charset=utf-8"
		} else if strings.HasSuffix(path, ".js") {
			contentType = "application/javascript; charset=utf-8"
		} else if strings.HasSuffix(path, ".svg") {
			contentType = "image/svg+xml"
		} else if strings.HasSuffix(path, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
			contentType = "image/jpeg"
		}

		c.Set("Content-Type", contentType)
		c.Set("Cache-Control", "public, max-age=86400")

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return c.Next()
		}

		return c.Send(data)
	}
}

// customErrorHandler handles HTTP errors
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	if code == fiber.StatusNotFound {
		return c.Status(code).SendString("404 - Sayfa bulunamadı")
	}

	log.Printf("Error %d: %v", code, err)
	return c.Status(code).SendString("Sunucu hatası")
}
