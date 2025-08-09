package sourcemap

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jsh-team/jshunter/internal/config"
	"github.com/jsh-team/jshunter/internal/utils/filesystem"
	"github.com/jsh-team/jshunter/internal/utils/logger"

	"github.com/pocketbase/pocketbase"
)

// saveSourceFile saves a source file directly to filesystem preserving original directory structure
func (p *SourcemapWorkerPool) saveSourceFile(app *pocketbase.PocketBase, domain string, sourceFile SourceFile, jsFileID string) error {
	// Get JS file record to get the hash
	jsFileRecord, err := app.FindRecordById("js_files", jsFileID)
	if err != nil {
		return fmt.Errorf("sourcemap worker: error finding JS file record: %w", err)
	}

	// Get the JS file hash
	jsFileHash := jsFileRecord.GetString("hash")
	if jsFileHash == "" {
		return fmt.Errorf("sourcemap worker: JS file has no hash")
	}

	// Clean the original path but preserve directory structure
	cleanedPath := filesystem.CleanSourcePath(sourceFile.Path)

	// Create the full directory structure: domain/js_hash/original/path/to/file
	sourceFileDir := filepath.Join(config.GetFilesPath(), domain, jsFileHash, "original", filepath.Dir(cleanedPath))
	if err := os.MkdirAll(sourceFileDir, 0755); err != nil {
		return fmt.Errorf("failed to create source directory %s: %w", sourceFileDir, err)
	}

	// Use the original filename (cleaned)
	filename := filepath.Base(cleanedPath)
	if filename == "" || filename == "." {
		filename = "unknown.js"
	}

	// Ensure we have a reasonable extension
	if filepath.Ext(filename) == "" {
		filename += ".js"
	}

	fullPath := filepath.Join(sourceFileDir, filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		logger.Debug("Source file already exists: %s", fullPath)
		return nil
	}

	// Write source file
	if err := os.WriteFile(fullPath, []byte(sourceFile.Content), 0644); err != nil {
		return fmt.Errorf("failed to write source file %s: %w", fullPath, err)
	}

	logger.Debug("Saved source file: %s", fullPath)
	return nil
}
