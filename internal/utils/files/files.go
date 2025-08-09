package files

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jsh-team/jshunter/internal/utils/logger"
)

// MoveTargetFiles moves all files from source to destination directory
func MoveTargetFiles(sourceDir, destDir string) error {
	// Check if source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		logger.Info("Source directory %s does not exist, nothing to move", sourceDir)
		return nil
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move db directory
	sourceDbDir := filepath.Join(sourceDir, "db")
	destDbDir := filepath.Join(destDir, "db")

	if _, err := os.Stat(sourceDbDir); err == nil {
		logger.Info("Moving database files from %s to %s", sourceDbDir, destDbDir)
		if err := MoveDirectory(sourceDbDir, destDbDir); err != nil {
			return fmt.Errorf("failed to move db directory: %w", err)
		}
	}

	// Move files directory
	sourceFilesDir := filepath.Join(sourceDir, "files")
	destFilesDir := filepath.Join(destDir, "files")

	if _, err := os.Stat(sourceFilesDir); err == nil {
		logger.Info("Moving files from %s to %s", sourceFilesDir, destFilesDir)
		if err := MoveDirectory(sourceFilesDir, destFilesDir); err != nil {
			return fmt.Errorf("failed to move files directory: %w", err)
		}
	}

	// Remove source directory if it's empty
	if isEmpty, err := IsDirEmpty(sourceDir); err == nil && isEmpty {
		logger.Info("Removing empty source directory: %s", sourceDir)
		os.Remove(sourceDir)
	}

	return nil
}

// MoveDirectory moves a directory from source to destination
func MoveDirectory(source, dest string) error {
	// Try to rename first (fastest if on same filesystem)
	if err := os.Rename(source, dest); err == nil {
		return nil
	}

	// If rename fails, copy and remove
	if err := CopyDirectory(source, dest); err != nil {
		return err
	}

	return os.RemoveAll(source)
}

// CopyDirectory recursively copies a directory
func CopyDirectory(source, dest string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return CopyFile(path, destPath)
	})
}

// CopyFile copies a single file
func CopyFile(source, dest string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = sourceFile.WriteTo(destFile)
	return err
}

// IsDirEmpty checks if a directory is empty
func IsDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil // Directory has at least one entry
	}

	return true, nil // Directory is empty
}

// IsValidPath checks if a path exists
func IsValidPath(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	return nil
}
