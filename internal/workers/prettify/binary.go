package prettify

import (
	"fmt"
	"os"
	"os/exec"

	"jshunter/internal/config"
	"jshunter/internal/utils/logger"
)

// getPrettierBinaryPath gets the prettifier binary path from configuration
func (p *PrettifyWorkerPool) getPrettierBinaryPath() (string, error) {
	// Use the prettifier path from configuration
	prettifierPath := config.PrettifierBinaryPath
	if prettifierPath == "" {
		return "", fmt.Errorf("prettifier binary path not configured")
	}

	// Check if the binary exists
	if _, err := os.Stat(prettifierPath); os.IsNotExist(err) {
		return "", fmt.Errorf("prettifier binary not found at: %s", prettifierPath)
	}

	return prettifierPath, nil
}

// prettifyFileInPlace prettifies a file in place by calling the prettier binary directly
func (p *PrettifyWorkerPool) prettifyFile(filePath string, fileType string) error {
	// Get the path to the prettier binary
	prettierPath, err := p.getPrettierBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find prettier binary: %w", err)
	}

	// Check if input file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", filePath)
	}

	// Run prettier with just the file path - it auto-detects the type
	cmd := exec.Command(prettierPath, "--"+fileType, filePath)

	_, err = cmd.Output()
	if err != nil {
		logger.Error("Prettifier command failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			logger.Error("Prettifier stderr: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("prettier formatting failed: %w", err)
	}

	return nil
}
