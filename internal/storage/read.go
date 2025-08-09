package storage

import (
	"fmt"
	"github.com/jsh-team/jshunter/internal/config"
	"github.com/jsh-team/jshunter/internal/utils/filesystem"
	urlutils "github.com/jsh-team/jshunter/internal/utils/url"
	"path/filepath"
)

// GetHTMLFilePath returns the absolute file path for an HTML file given URL and hash
func GetHTMLFilePath(fileURL, hash string) (string, error) {
	domain, err := filesystem.ExtractDomain(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract domain from URL %s: %w", fileURL, err)
	}
	filename, err := urlutils.GetFileNameFromUrl(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract filename from URL %s: %w", fileURL, err)
	}

	// Return absolute path using config
	return filepath.Join(config.GetFilesPath(), domain, hash, filename), nil
}

// GetJSFilePath returns the absolute file path for a JavaScript file given URL and hash
func GetJSFilePath(fileURL, hash string) (string, error) {
	domain, err := filesystem.ExtractDomain(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract domain from URL %s: %w", fileURL, err)
	}
	filename, err := urlutils.GetFileNameFromUrl(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract filename from URL %s: %w", fileURL, err)
	}
	// Return absolute path using config
	return filepath.Join(config.GetFilesPath(), domain, hash, filename), nil
}
