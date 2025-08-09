package analysis

import (
	"fmt"
	"time"

	"github.com/JSH-Team/JSHunter/internal/storage"
	"github.com/JSH-Team/JSHunter/internal/utils/logger"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processJob processes a single analysis job
func (p *AnalysisWorkerPool) processJob(workerID int, job AnalysisJob) {
	startTime := time.Now()
	errorCount := 0
	jsFileRecord := job.Record

	// Get file hash and URL to build the path
	bodyHash := jsFileRecord.GetString("hash")
	fileURL := jsFileRecord.GetString("url")
	if bodyHash == "" || fileURL == "" {
		errorCount++
		logger.Error("Analysis Worker %d failed: missing hash or URL for record %s", workerID, jsFileRecord.Id)
		jsFileRecord.Set("analysis_status", "failed")
		job.App.Save(jsFileRecord)
		logger.Info("Analysis worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Get JS file path using filesystem utility
	fullPath, err := storage.GetJSFilePath(fileURL, bodyHash)
	if err != nil {
		errorCount++
		logger.Error("Analysis Worker %d failed to get file path for %s: %v", workerID, fileURL, err)
		jsFileRecord.Set("analysis_status", "failed")
		job.App.Save(jsFileRecord)
		logger.Info("Analysis worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Analyze JavaScript file directly using the integrated analyzer
	findings, err := AnalyzeFile(fullPath)
	if err != nil {
		errorCount++
		logger.Error("Analysis Worker %d failed to analyze file %s: %v", workerID, fullPath, err)
		jsFileRecord.Set("analysis_status", "failed")
		job.App.Save(jsFileRecord)
		logger.Info("Analysis worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Save findings to database
	_, err = p.saveFindings(job.App, jsFileRecord.Id, findings)
	if err != nil {
		errorCount++
		logger.Error("Analysis Worker %d failed to save findings for %s: %v", workerID, jsFileRecord.GetString("url"), err)
		jsFileRecord.Set("analysis_status", "failed")
		job.App.Save(jsFileRecord)
		logger.Info("Analysis worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Update final status
	jsFileRecord.Set("analysis_status", "processed")
	if err := job.App.Save(jsFileRecord); err != nil {
		errorCount++
		logger.Error("Analysis Worker %d failed to save final record for %s: %v", workerID, jsFileRecord.GetString("url"), err)
	}

}

// saveFindings saves analysis findings to the database
func (p *AnalysisWorkerPool) saveFindings(app *pocketbase.PocketBase, jsFileID string, findings []Finding) (int, error) {
	if len(findings) == 0 {
		return 0, nil
	}

	findingsCollection, err := app.FindCollectionByNameOrId("findings")
	if err != nil {
		return 0, fmt.Errorf("error fetching findings collection: %w", err)
	}

	savedCount := 0
	now := time.Now()

	for _, finding := range findings {
		// Create finding record
		newRecord := core.NewRecord(findingsCollection)
		newRecord.Set("type", finding.Type)
		newRecord.Set("line", finding.Line)
		newRecord.Set("column", finding.Column)
		newRecord.Set("value", finding.Value)
		newRecord.Set("js_file", jsFileID)
		newRecord.Set("metadata", finding.Data)
		newRecord.Set("created_at", now)

		if err := app.Save(newRecord); err != nil {
			// Log error but continue with other findings
			logger.Error("Error saving finding: %v", err)
			continue
		}
		savedCount++
	}

	return savedCount, nil
}
