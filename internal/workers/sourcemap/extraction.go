package sourcemap

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JSH-Team/JSHunter/internal/utils/fetch"
	"github.com/JSH-Team/JSHunter/internal/utils/hash"
	"github.com/JSH-Team/JSHunter/internal/utils/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// SourceMap represents the structure of a JavaScript source map
type SourceMap struct {
	Version        int      `json:"version"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
}

// SourceMapResult contains the result of sourcemap processing
type SourceMapResult struct {
	Found       bool
	TempDir     string
	SourceFiles []SourceFile
}

// SourceFile represents an extracted source file
type SourceFile struct {
	Path         string // Relative path within temp directory
	OriginalPath string // Original path from sourcemap
	Content      string
}

// ProcessSourceMap is the main function that handles all sourcemap extraction logic
func ProcessSourceMap(jsBody string, jsURL string) (SourceMapResult, error) {
	result := SourceMapResult{
		Found:       false,
		SourceFiles: []SourceFile{},
	}

	// Step 1: Check for sourcemap URL in JavaScript body
	sourceMapURL := findSourceMapURL(jsBody)

	var sourceMapContent []byte
	var err error

	if sourceMapURL != "" {
		// Step 2a: Process sourcemap URL (data URI or regular URL)
		sourceMapContent, err = getSourceMapContent(sourceMapURL, jsURL)
		if err != nil {
			// Step 2b: If failed, try fallback .map URL
			sourceMapContent, err = tryFallbackMapURL(jsURL)
		}
	} else {
		// Step 2b: No sourcemap URL found, try fallback .map URL
		sourceMapContent, err = tryFallbackMapURL(jsURL)
	}

	if err != nil || sourceMapContent == nil {
		return result, nil // No sourcemap found, not an error
	}

	// Step 3: Parse sourcemap and extract sources to temp directory
	tempDir, sourceFiles, err := extractSourcesToTempDir(sourceMapContent, jsURL)
	if err != nil {
		return result, fmt.Errorf("failed to extract sources: %w", err)
	}

	result.Found = true
	result.TempDir = tempDir
	result.SourceFiles = sourceFiles

	return result, nil
}

// findSourceMapURL looks for sourcemap URL in JavaScript content
func findSourceMapURL(jsBody string) string {
	// Look for sourcemap URL comment: //# sourceMappingURL=...
	re := regexp.MustCompile(`//[@#]\s*sourceMappingURL=(.*)`)
	matches := re.FindAllStringSubmatch(jsBody, -1)

	if len(matches) > 0 {
		// Use the last match (as per specification)
		sourceMapURL := strings.TrimSpace(matches[len(matches)-1][1])
		return sourceMapURL
	}

	return ""
}

// getSourceMapContent retrieves sourcemap content from URL or data URI
func getSourceMapContent(sourceMapURL string, jsURL string) ([]byte, error) {
	// Handle inline data URI sourcemaps
	if strings.HasPrefix(sourceMapURL, "data:") {
		return url.DecodeDataURI(sourceMapURL)
	}

	// Convert relative URLs to absolute
	fullURL, err := url.ToAbsoluteURL(jsURL, sourceMapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute URL: %w", err)
	}

	// Fetch the sourcemap from the URL
	return fetchSourceMapContent(fullURL)
}

// tryFallbackMapURL tries to fetch sourcemap using .map extension
func tryFallbackMapURL(jsURL string) ([]byte, error) {
	// Remove query string and add .map extension
	cleanURL, err := url.RemoveQueryString(jsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to remove query string from URL: %w", err)
	}

	mapURL := cleanURL + ".map"
	return fetchSourceMapContent(mapURL)
}

// fetchSourceMapContent downloads sourcemap content using the fetch utility
func fetchSourceMapContent(mapURL string) ([]byte, error) {
	assetFetcher := fetch.NewAssetFetcher()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content, success, err := assetFetcher.RateLimitedGet(ctx, mapURL)
	if !success || err != nil {
		return nil, fmt.Errorf("failed to download sourcemap from %s: %w", mapURL, err)
	}

	// Validate that the content is actually a sourcemap before returning
	if !isValidSourceMapContent([]byte(content)) {
		return nil, fmt.Errorf("downloaded content is not a valid sourcemap")
	}

	return []byte(content), nil
}

// isValidSourceMapContent checks if the content looks like a valid sourcemap
func isValidSourceMapContent(content []byte) bool {
	// Must be valid JSON
	var temp map[string]interface{}
	if err := json.Unmarshal(content, &temp); err != nil {
		return false
	}

	// Must have required sourcemap fields
	version, hasVersion := temp["version"]
	sources, hasSources := temp["sources"]

	// Check version is a number (typically 3)
	if !hasVersion {
		return false
	}
	if versionNum, ok := version.(float64); !ok || versionNum < 1 {
		return false
	}

	// Must have sources array
	if !hasSources {
		return false
	}
	if _, ok := sources.([]interface{}); !ok {
		return false
	}

	return true
}

// extractSourcesToTempDir parses sourcemap and extracts sources to temporary directory
func extractSourcesToTempDir(sourceMapContent []byte, jsURL string) (string, []SourceFile, error) {
	var sourceMap SourceMap

	// Parse sourcemap JSON
	if err := json.Unmarshal(sourceMapContent, &sourceMap); err != nil {
		return "", nil, fmt.Errorf("failed to parse sourcemap JSON: %w", err)
	}

	// Verify sourcemap has content to extract
	if len(sourceMap.Sources) == 0 || len(sourceMap.SourcesContent) == 0 {
		return "", nil, fmt.Errorf("sourcemap contains no sources or content")
	}

	// Create temporary directory for extracted source files
	jsHash := hash.GenerateSha256Hash(jsURL)
	tempDir := filepath.Join(os.TempDir(), "sourcemaps", jsHash)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	var sourceFiles []SourceFile

	// Extract each source file
	for i, sourcePath := range sourceMap.Sources {
		if i >= len(sourceMap.SourcesContent) {
			continue // Skip if no content available
		}

		// Sanitize the source path
		cleanPath := sanitizeSourcePath(sourcePath)

		// Build full file path within temp directory
		fullPath := filepath.Join(tempDir, cleanPath)

		// Create necessary directories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create directory %s: %w", filepath.Dir(fullPath), err)
		}

		// Write source content to file
		content := sourceMap.SourcesContent[i]
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", nil, fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}

		// Add to source files list
		sourceFiles = append(sourceFiles, SourceFile{
			Path:         cleanPath,
			OriginalPath: sourcePath,
			Content:      content,
		})
	}

	return tempDir, sourceFiles, nil
}

// sanitizeSourcePath cleans and sanitizes source file paths
func sanitizeSourcePath(sourcePath string) string {
	// Remove leading slashes and dots to prevent path traversal
	cleanPath := strings.TrimLeft(sourcePath, "./")

	// Replace path separators for consistency
	cleanPath = filepath.Clean(cleanPath)

	// Sanitize for Windows if needed
	if runtime.GOOS == "windows" {
		cleanPath = sanitizeWindowsPath(cleanPath)
	}

	return cleanPath
}

// sanitizeWindowsPath removes illegal characters from Windows file paths
func sanitizeWindowsPath(path string) string {
	re := regexp.MustCompile(`[?%*|:"<>]`)
	return re.ReplaceAllString(path, "-")
}

// CleanupTempDir removes the temporary directory and all its contents
func CleanupTempDir(tempDir string) error {
	if tempDir == "" {
		return nil
	}
	return os.RemoveAll(tempDir)
}
