package extraction

import (
	"context"
	"fmt"
	"time"

	"jshunter/internal/config"
	"jshunter/internal/storage"
	"jshunter/internal/utils/db"
	"jshunter/internal/utils/hash"
	"jshunter/internal/utils/logger"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// processJob processes a single extraction job
func (p *ExtractionWorkerPool) processJob(workerID int, job ExtractionJob) {
	startTime := time.Now()
	errorCount := 0

	logger.Info("Extraction Worker %d started processing", workerID)

	// Create job-specific context with timeout
	jobCtx, cancel := context.WithTimeout(job.Context, time.Duration(config.BrowserWorkerTimeout)*time.Second)
	defer cancel()

	// Process desktop extraction
	html, jsFiles, err := p.processEndpointWithBrowser(jobCtx, job.Record, false)
	if err != nil {
		errorCount++
		logger.Error("Extraction Worker %d failed to process endpoint %s: %v", workerID, job.Record.GetString("url"), err)
		// Mark as failed
		job.Record.Set("extraction_status", "failed")
		job.App.Save(job.Record)
		logger.Info("Extraction worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// Save desktop results to database
	if err := p.saveProcessingResults(job.App, job.Record, html, jsFiles, false); err != nil {
		errorCount++
		logger.Error("Extraction Worker %d failed to save results for %s: %v", workerID, job.Record.GetString("url"), err)
		// Mark as failed
		job.Record.Set("extraction_status", "failed")
		job.App.Save(job.Record)
		logger.Info("Extraction worker finished in %v with %d errors", time.Since(startTime), errorCount)
		return
	}

	// If mobile extraction is enabled, do mobile extraction too
	if config.MobileExtractionEnabled {
		mobileHTML, mobileJSFiles, _ := p.processEndpointWithBrowser(jobCtx, job.Record, true)

		if err := p.saveProcessingResults(job.App, job.Record, mobileHTML, mobileJSFiles, true); err != nil {
			errorCount++
			logger.Error("Extraction Worker %d failed to save mobile results for %s: %v", workerID, job.Record.GetString("url"), err)
			job.Record.Set("extraction_status", "failed")
			job.App.Save(job.Record)
			logger.Info("Extraction worker finished in %v with %d errors", time.Since(startTime), errorCount)
			return
		}

	}

	// Mark as successfully processed
	job.Record.Set("extraction_status", "processed")
	if err := job.App.Save(job.Record); err != nil {
		errorCount++
		logger.Error("Extraction Worker %d failed to save final record for %s: %v", workerID, job.Record.GetString("url"), err)
	}

	logger.Info("Extraction worker finished in %v with %d errors", time.Since(startTime), errorCount)
}

// processEndpointWithBrowser handles the actual browser processing
func (p *ExtractionWorkerPool) processEndpointWithBrowser(_ context.Context, record *core.Record, isMobile bool) (string, []JSFileResult, error) {
	endpointURL := record.GetString("url")

	// Extract headers from record
	headersMap := make(map[string]string)
	if rawHeaders := record.Get("request_headers"); rawHeaders != nil {
		if headers, ok := rawHeaders.(types.JSONRaw); ok {
			if convertedHeaders, err := db.ConvertHeadersToMap(headers); err == nil {
				headersMap = convertedHeaders
			}
		}
	}

	// Create browser extractor
	extractor := NewBrowserExtractor()
	defer extractor.Close()

	// Create browser options
	browserOptions := ExtractionOptions{
		Headers:     headersMap,
		Mobile:      isMobile,
		Timeout:     60 * time.Second,
		PageTimeout: 15 * time.Second, // Timeout más corto para evitar problemas como AnimeFlv
	}

	// Extract HTML and JS for the specified version (desktop or mobile)
	html, loadedJS, err := extractor.ExtractJavaScript(endpointURL, browserOptions)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract HTML: %w", err)
	}

	var jsFiles []JSFileResult

	// Convert loaded JS to results
	for _, resource := range loadedJS {
		jsType := "normal"
		if isMobile {
			jsType = "mobile"
		}
		if resource.Source == "inline" {
			jsType = "inline"
		}

		jsFiles = append(jsFiles, JSFileResult{
			URL:     resource.URL,
			Content: resource.Content,
			Type:    jsType,
		})
	}

	// Los scripts inline ya están incluidos en loadedJS por el nuevo extractor

	return html, jsFiles, nil
}

// saveProcessingResults saves the extracted HTML and JavaScript files
func (p *ExtractionWorkerPool) saveProcessingResults(app *pocketbase.PocketBase, endpointRecord *core.Record, html string, jsFiles []JSFileResult, isMobile bool) error {
	// Save HTML file and calculate structural hash
	htmlHash := storage.SaveHTMLFile(endpointRecord.GetString("url"), html)
	if htmlHash != "" {
		if isMobile {
			endpointRecord.Set("mobile_hash", htmlHash)
		} else {
			endpointRecord.Set("hash", htmlHash)
		}
	}

	// Save JavaScript files directly
	var jsFileIDs []string
	jsFileCollection, err := app.FindCollectionByNameOrId("js_files")
	if err != nil {
		logger.Error("Error fetching js files collection: %v", err)
		return err
	}

	for _, jsFile := range jsFiles {
		// Check for duplicates before saving
		contentHash := hash.GenerateSha256Hash(jsFile.Content)
		if existingID := checkExistingJSFile(app, jsFile.URL, contentHash); existingID != "" {
			jsFileIDs = append(jsFileIDs, existingID)
			continue
		}
		storage.SaveJSFile(jsFile.URL, jsFile.Content)

		newRecord := core.NewRecord(jsFileCollection)
		newRecord.Set("url", jsFile.URL)
		newRecord.Set("hash", contentHash)
		newRecord.Set("type", jsFile.Type)
		app.Save(newRecord)
		jsFileIDs = append(jsFileIDs, newRecord.Id)
	}

	// Update endpoint record with file information
	endpointRecord.Set("js_files", jsFileIDs)

	return app.Save(endpointRecord)
}

func checkExistingJSFile(app *pocketbase.PocketBase, url string, contentHash string) string {
	existingRecord, err := app.FindFirstRecordByFilter(
		"js_files",
		"url = {:url} || hash = {:hash}",
		map[string]any{"url": url, "hash": contentHash},
	)
	if err == nil && existingRecord != nil {
		return existingRecord.Id
	}
	return ""
}
