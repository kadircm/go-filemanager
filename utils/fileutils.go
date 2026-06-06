package utils

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
)

// FileCategory represents the type category of a file
type FileCategory string

const (
	CategoryFolder   FileCategory = "folder"
	CategoryImage    FileCategory = "image"
	CategoryVideo    FileCategory = "video"
	CategoryAudio    FileCategory = "audio"
	CategoryDocument FileCategory = "document"
	CategoryCode     FileCategory = "code"
	CategoryArchive  FileCategory = "archive"
	CategoryOther    FileCategory = "other"
)

// GetFileCategory determines the category of a file by its extension
func GetFileCategory(filename string) FileCategory {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".tiff":
		return CategoryImage
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v":
		return CategoryVideo
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".wma", ".m4a":
		return CategoryAudio
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf", ".csv", ".odt":
		return CategoryDocument
	case ".go", ".py", ".js", ".ts", ".html", ".css", ".json", ".xml", ".yaml", ".yml",
		".java", ".c", ".cpp", ".h", ".rs", ".rb", ".php", ".sh", ".bash", ".sql",
		".md", ".toml", ".ini", ".conf", ".cfg", ".env", ".dockerfile", ".vue", ".jsx", ".tsx",
		".swift", ".kt", ".scala", ".r", ".lua", ".pl", ".ex", ".exs":
		return CategoryCode
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".tgz":
		return CategoryArchive
	default:
		return CategoryOther
	}
}

// GetMimeType returns the MIME type of a file
func GetMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// IsTextFile checks if a file is likely a text file
func IsTextFile(filename string) bool {
	cat := GetFileCategory(filename)
	return cat == CategoryCode || cat == CategoryDocument ||
		strings.ToLower(filepath.Ext(filename)) == ".txt" ||
		strings.ToLower(filepath.Ext(filename)) == ".log" ||
		strings.ToLower(filepath.Ext(filename)) == ".csv" ||
		strings.ToLower(filepath.Ext(filename)) == ".md"
}

// IsMediaFile checks if a file is a media file (video/audio/image)
func IsMediaFile(filename string) bool {
	cat := GetFileCategory(filename)
	return cat == CategoryImage || cat == CategoryVideo || cat == CategoryAudio
}

// FormatFileSize converts bytes to human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetCodeMirrorMode returns the CodeMirror language mode for a file extension
func GetCodeMirrorMode(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	modes := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "jsx",
		".tsx":        "tsx",
		".html":       "html",
		".htm":        "html",
		".css":        "css",
		".scss":       "css",
		".json":       "json",
		".xml":        "xml",
		".yaml":       "yaml",
		".yml":        "yaml",
		".md":         "markdown",
		".sql":        "sql",
		".sh":         "shell",
		".bash":       "shell",
		".php":        "php",
		".java":       "java",
		".c":          "cpp",
		".cpp":        "cpp",
		".h":          "cpp",
		".rs":         "rust",
		".rb":         "ruby",
		".lua":        "lua",
		".r":          "r",
		".toml":       "toml",
		".ini":        "ini",
		".conf":       "nginx",
		".dockerfile": "dockerfile",
		".vue":        "vue",
		".swift":      "swift",
		".kt":         "kotlin",
	}

	if mode, ok := modes[ext]; ok {
		return mode
	}
	return "plaintext"
}
