package dechunker

import (
	"bufio"
	"fmt"
	"github.com/JSH-Team/JSHunter/internal/config"
	"os"
	"os/exec"
	"strings"
)

// Dechunker wraps the dechunker executable
type Dechunker struct {
	dechunkerPath string
}

// NewDechunker creates a new dechunker instance
func NewDechunker() (*Dechunker, error) {
	// Use the dechunker path from configuration
	dechunkerPath := config.DechunkerBinaryPath
	if dechunkerPath == "" {
		return nil, fmt.Errorf("dechunker binary path not configured")
	}

	// Check if the binary exists
	if _, err := os.Stat(dechunkerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("dechunker binary not found at: %s", dechunkerPath)
	}

	return &Dechunker{
		dechunkerPath: dechunkerPath,
	}, nil
}

// ExtractChunks performs chunk extraction on a JavaScript file using the dechunker
func (d *Dechunker) ExtractChunks(filePath string, baseURL string) ([]ChunkURL, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Run the dechunker with the file path and base URL
	cmd := exec.Command(d.dechunkerPath, filePath, "--url", baseURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run dechunker: %w", err)
	}

	// Parse line-by-line URLs
	var chunkURLs []ChunkURL
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		// Create ChunkURL from the line (URL)
		chunkURLs = append(chunkURLs, ChunkURL{
			URL:      line,
			Type:     "unknown", // We don't know the type from line output
			ChunkID:  "",        // We don't have chunk ID from line output
			Metadata: make(map[string]interface{}),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse dechunker output: %w", err)
	}

	return chunkURLs, nil
}

// ExtractChunksFromFile is the main entry point for extracting chunks from a JavaScript file
func ExtractChunksFromFile(filePath string, baseURL string) ([]ChunkURL, error) {
	dechunker, err := NewDechunker()
	if err != nil {
		return nil, fmt.Errorf("failed to create dechunker: %w", err)
	}

	return dechunker.ExtractChunks(filePath, baseURL)
}
