package sourcemap

import (
	"os"

	"github.com/JSH-Team/JSHunter/internal/storage"
	"github.com/JSH-Team/JSHunter/internal/utils/filesystem"
)

// processJob processes a single sourcemap job
func (p *SourcemapWorkerPool) processJob(workerID int, job SourcemapJob) {
	jsFileRecord := job.Record

	// Get file hash and URL to build the path
	bodyHash := jsFileRecord.GetString("hash")
	fileURL := jsFileRecord.GetString("url")
	if bodyHash == "" || fileURL == "" {
		jsFileRecord.Set("sourcemap_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Read JS file content directly from filesystem using filesystem utility
	filePath, err := storage.GetJSFilePath(fileURL, bodyHash)
	if err != nil {
		jsFileRecord.Set("sourcemap_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Read file content
	jsContentBytes, err := os.ReadFile(filePath)
	if err != nil {
		jsFileRecord.Set("sourcemap_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}
	jsContent := string(jsContentBytes)

	// Extract domain for organizing source files
	domain, err := filesystem.ExtractDomain(jsFileRecord.GetString("url"))
	if err != nil {
		jsFileRecord.Set("sourcemap_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Process sourcemap
	result, err := ProcessSourceMap(jsContent, jsFileRecord.GetString("url"))
	if err != nil {
		// Not having a sourcemap is expected and not an error, so we don't log this as an error
		jsFileRecord.Set("sourcemap_status", "processed")
		job.App.Save(jsFileRecord)
		return
	}

	// Save source files to filesystem directly
	successCount := 0
	for _, sourceFile := range result.SourceFiles {
		if err := p.saveSourceFile(job.App, domain, sourceFile, jsFileRecord.Id); err != nil {
			continue // Continue with other files
		}
		successCount++
	}

	// Update final status
	jsFileRecord.Set("sourcemap_status", "processed")

	if err := job.App.Save(jsFileRecord); err != nil {
		return
	}

}
