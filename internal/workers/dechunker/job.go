package dechunker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JSH-Team/JSHunter/internal/storage"
	"github.com/JSH-Team/JSHunter/internal/utils/fetch"
	"github.com/JSH-Team/JSHunter/internal/utils/logger"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processJob processes a single dechunker job
func (p *DechunkerWorkerPool) processJob(workerID int, job DechunkerJob) {
	errorCount := 0
	jsFileRecord := job.Record

	// Get file hash and URL to build the path
	bodyHash := jsFileRecord.GetString("hash")
	fileURL := jsFileRecord.GetString("url")
	if bodyHash == "" || fileURL == "" {
		errorCount++
		logger.Error("Dechunker Worker %d failed: missing hash or URL for record %s", workerID, jsFileRecord.Id)
		jsFileRecord.Set("dechunker_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Get JS file path using filesystem utility
	fullPath, err := storage.GetJSFilePath(fileURL, bodyHash)
	if err != nil {
		errorCount++
		logger.Error("Dechunker Worker %d failed to get file path for %s: %v", workerID, fileURL, err)
		jsFileRecord.Set("dechunker_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Extract chunks from JavaScript file
	chunkURLs, err := ExtractChunksFromFile(fullPath, fileURL)
	if err != nil {
		errorCount++
		logger.Error("Dechunker Worker %d failed to extract chunks from file %s: %v", workerID, fullPath, err)
		jsFileRecord.Set("dechunker_status", "failed")
		job.App.Save(jsFileRecord)
		return
	}

	// Process chunk URLs - fetch and save as JS files
	if len(chunkURLs) > 0 {
		logger.Info("Found %d potential chunk URLs for %s", len(chunkURLs), fileURL)
		jsFileRecord.Set("has_chunks", true)
		job.App.Save(jsFileRecord)
		err = p.fetchAndSaveChunks(job.App, jsFileRecord.Id, chunkURLs)
		if err != nil {
			errorCount++
			logger.Error("Dechunker Worker %d failed to fetch and save chunks for %s: %v", workerID, jsFileRecord.GetString("url"), err)
		}

		// Set has_chunks flag if we found any chunks
	}

	// Always mark as processed (even if no chunks found)
	jsFileRecord.Set("dechunker_status", "processed")
	jsFileRecord.Set("last_modified", time.Now())
	if err := job.App.Save(jsFileRecord); err != nil {
		errorCount++
		logger.Error("Dechunker Worker %d failed to save final record for %s: %v", workerID, jsFileRecord.GetString("url"), err)
	}
}

// fetchAndSaveChunks fetches chunk URLs and saves them as JS files
func (p *DechunkerWorkerPool) fetchAndSaveChunks(app *pocketbase.PocketBase, parentJSFileID string, chunkURLs []ChunkURL) error {
	if len(chunkURLs) == 0 {
		return nil
	}

	jsFilesCollection, err := app.FindCollectionByNameOrId("js_files")
	if err != nil {
		return fmt.Errorf("error fetching js_files collection: %w", err)
	}

	// Create rate-limited fetcher
	fetcher := fetch.NewAssetFetcher()
	now := time.Now()

	for _, chunkURL := range chunkURLs {
		// Use the URL directly from the binary (already resolved)
		absoluteURL := chunkURL.URL

		// Check if this chunk already exists in the database (by URL)
		existingRecord, err := app.FindFirstRecordByFilter(
			"js_files",
			"url = {:url}",
			map[string]any{"url": absoluteURL},
		)
		if err == nil && existingRecord != nil {
			continue
		}
		// Fetch chunk content with rate limiting
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		content, contentType, success, err := fetcher.RateLimitedGetWithContentType(ctx, absoluteURL)
		cancel()

		if err != nil || !success {
			logger.Error("Failed to fetch chunk %s: success=%v, err=%v", absoluteURL, success, err)
			continue
		}

		// Validate content type
		if !strings.Contains(contentType, "javascript") && !strings.Contains(contentType, "text/plain") {
			logger.Debug("Skipping chunk %s with incorrect content type: %s", absoluteURL, contentType)
			continue
		}

		// Content sniffing for HTML
		if strings.HasPrefix(strings.TrimSpace(content), "<!DOCTYPE html>") || strings.HasPrefix(strings.TrimSpace(content), "<html>") {
			logger.Debug("Skipping chunk %s because it appears to be HTML", absoluteURL)
			continue
		}

		if len(content) == 0 {
			logger.Error("Failed to fetch chunk %s: empty content", absoluteURL)
			continue
		}

		// Save content to filesystem
		hash := storage.SaveJSFile(absoluteURL, content)
		// Create JS file record for the chunk
		newRecord := core.NewRecord(jsFilesCollection)
		newRecord.Set("url", absoluteURL)
		newRecord.Set("hash", hash)
		newRecord.Set("parent_id", parentJSFileID)
		newRecord.Set("type", "chunk")
		newRecord.Set("has_chunks", false) // Chunks themselves don't have chunks
		newRecord.Set("created_at", now)

		if err := app.Save(newRecord); err != nil {
			logger.Error("Error saving chunk JS file record for %s: %v", absoluteURL, err)
			continue
		}

	}

	return nil
}
