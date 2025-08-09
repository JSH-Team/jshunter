package storage

import (
	"github.com/JSH-Team/JSHunter/internal/config"
	"github.com/JSH-Team/JSHunter/internal/utils/filesystem"
	"github.com/JSH-Team/JSHunter/internal/utils/hash"
	"github.com/JSH-Team/JSHunter/internal/utils/html"
	"github.com/JSH-Team/JSHunter/internal/utils/logger"
	urlutils "github.com/JSH-Team/JSHunter/internal/utils/url"
	"os"
	"path/filepath"
)

// saveJSFile saves JavaScript content directly to filesystem
func SaveJSFile(url string, content string) string {
	// Generate content hash for JS files
	contentHash := hash.GenerateSha256Hash(content)

	// Extract domain from URL
	domain, err := filesystem.ExtractDomain(url)
	if err != nil {
		logger.Error("Failed to extract domain from JS URL %s: %v", url, err)
		return ""
	}

	// Create domain directory
	domainDir := filepath.Join(config.GetFilesPath(), domain)
	storageDir := filepath.Join(domainDir, contentHash)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		logger.Error("Failed to create domain directory %s: %v", domainDir, err)
		return ""
	}

	// Create JS filename and path
	filename, err := urlutils.GetFileNameFromUrl(url)
	if err != nil {
		logger.Error("Failed to extract filename from URL %s: %v", url, err)
		return ""
	}
	fullPath := filepath.Join(storageDir, filename)

	// Write JS file if it doesn't exist
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			logger.Error("Failed to write JS file %s: %v", fullPath, err)
			return ""
		}
	}

	return contentHash
}

func SaveHTMLFile(url string, content string) string {
	hash, err := html.GenerateHTMLHash(content)
	if err != nil {
		logger.Error("Failed to calculate structural hash for %s: %v", url, err)
		return ""
	}

	// Extract domain from URL
	domain, err := filesystem.ExtractDomain(url)
	if err != nil {
		logger.Error("Failed to extract domain from URL %s: %v", url, err)
		return ""
	}

	// Create domain directory
	domainDir := filepath.Join(config.GetFilesPath(), domain)
	storageDir := filepath.Join(domainDir, hash)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		logger.Error("Failed to create domain directory %s: %v", domainDir, err)
		return ""
	}

	// Create filename and path using just the structural hash
	filename, err := urlutils.GetFileNameFromUrl(url)
	if err != nil {
		logger.Error("Failed to extract filename from URL %s: %v", url, err)
		return ""
	}
	fullPath := filepath.Join(storageDir, filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return hash // File already exists, return the hash
	}

	// Write HTML file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		logger.Error("Failed to write HTML file %s: %v", fullPath, err)
		return ""
	}

	return hash
}
