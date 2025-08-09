package prettify

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"jshunter/internal/utils/logger"
)

// processJob processes a single prettify job
func (p *PrettifyWorkerPool) processJob(workerID int, job PrettifyJob) {
	startTime := time.Now()
	errorCount := 0

	// Get file path directly from job
	fullPath := job.FilePath
	if fullPath == "" {
		errorCount++
		logger.Error("Prettify Worker %d failed: missing file path for job", workerID)
		// Only set status if this is a real record (not temp record for HTML)
		if job.Record != nil && job.Record.Id != "" {
			job.Record.Set("prettify_status", "failed")
			job.App.Save(job.Record)
		}
		logger.Info("Prettify worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Call prettifier binary directly on the file
	if err := p.prettifyFile(fullPath, job.Type); err != nil {
		errorCount++

		logger.Error("Prettify Worker %d failed to prettify file %s: %v | URL: %s", workerID, fullPath, err, job.Record.Get("url"))
		// Only set status if this is a real record (not temp record for HTML)
		if job.Record != nil && job.Record.Id != "" {
			job.Record.Set("prettify_status", "failed")
			job.App.Save(job.Record)
		}
		logger.Info("Prettify worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Mark as successfully processed (only for real records, not temp HTML records)
	if job.Record != nil && job.Record.Id != "" {
		lines, err := countLines(fullPath)
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
		}

		job.Record.Set("line_count", lines)
		job.Record.Set("prettify_status", "processed")
		job.Record.Set("last_modified", time.Now())
		if err := job.App.Save(job.Record); err != nil {
			errorCount++
			logger.Error("Prettify Worker %d failed to save final record: %v", workerID, err)
		}
	}

}

func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	const bufferSize = 32 * 1024 // 32KB buffer
	buf := make([]byte, bufferSize)
	lineCount := 0
	lastCharIsNewline := true // Assume true for empty file or start of file

	for {
		c, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("error reading file: %w", err)
		}

		if c > 0 {
			lineCount += bytes.Count(buf[:c], []byte{'\n'})
			lastCharIsNewline = (buf[c-1] == '\n')
		}

		if err == io.EOF {
			break
		}
	}

	// If the file is not empty and doesn't end with a newline, the last line isn't counted.
	// So, we add 1 to the count.
	if !lastCharIsNewline {
		lineCount++
	}

	return lineCount, nil
}
