package filesystem

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractDomain extracts and cleans the domain from a URL for filesystem use
func ExtractDomain(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	domain := parsedURL.Hostname()
	if domain == "" {
		return "", fmt.Errorf("no hostname found in URL")
	}

	// Clean domain for filesystem use
	return CleanPath(domain), nil
}

// CleanPath cleans a path component for safe filesystem use
func CleanPath(path string) string {
	if path == "" {
		return "unknown"
	}

	// Remove protocol prefixes
	path = strings.TrimPrefix(path, "http://")
	path = strings.TrimPrefix(path, "https://")
	path = strings.TrimPrefix(path, "ftp://")

	// Replace invalid characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	path = invalidChars.ReplaceAllString(path, "_")

	// Replace multiple dots with single dot
	multipleDots := regexp.MustCompile(`\.{2,}`)
	path = multipleDots.ReplaceAllString(path, ".")

	// Trim dots and spaces from ends
	path = strings.Trim(path, ". ")

	// Replace spaces with underscores
	path = strings.ReplaceAll(path, " ", "_")

	// Ensure it's not empty or reserved
	if path == "" || path == "." || path == ".." {
		return "unknown"
	}

	// Limit length
	if len(path) > 100 {
		path = path[:100]
	}

	return path
}

// CleanSourcePath cleans a source file path for filesystem storage
func CleanSourcePath(path string) string {
	if path == "" {
		return "unknown.js"
	}

	// Remove leading slashes and protocol prefixes
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "../")

	// Split path into components and clean each
	components := strings.Split(path, "/")
	cleanedComponents := make([]string, 0, len(components))

	for _, component := range components {
		cleaned := CleanPathComponent(component)
		if cleaned != "" && cleaned != "." && cleaned != ".." {
			cleanedComponents = append(cleanedComponents, cleaned)
		}
	}

	if len(cleanedComponents) == 0 {
		return "unknown.js"
	}

	return filepath.Join(cleanedComponents...)
}

// CleanPathComponent cleans a single path component
func CleanPathComponent(component string) string {
	if component == "" {
		return ""
	}

	// Replace invalid characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	component = invalidChars.ReplaceAllString(component, "_")

	// Replace multiple dots with single dot
	multipleDots := regexp.MustCompile(`\.{2,}`)
	component = multipleDots.ReplaceAllString(component, ".")

	// Trim dots and spaces from ends
	component = strings.Trim(component, ". ")

	// Replace spaces with underscores
	component = strings.ReplaceAll(component, " ", "_")

	// Limit length
	if len(component) > 50 {
		component = component[:50]
	}

	return component
}
