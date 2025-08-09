package db

import (
	"jshunter/internal/storage"
	"jshunter/internal/utils/db"
	"jshunter/internal/utils/html"
	"jshunter/internal/utils/logger"
	"jshunter/internal/workers/analysis"
	"jshunter/internal/workers/dechunker"
	"jshunter/internal/workers/extraction"
	"jshunter/internal/workers/prettify"
	"jshunter/internal/workers/sourcemap"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterHooks registers all database hooks
func RegisterHooks(app *pocketbase.PocketBase) error {

	// =============================================================================
	// ENDPOINTS HOOKS
	// =============================================================================
	app.OnRecordAfterCreateSuccess("tmp_endpoints").BindFunc(func(e *core.RecordEvent) error {
		key := e.Record.BaseFilesPath() + "/" + e.Record.GetString("tmp_body")
		body, err := db.ReadFileFromRecord(app, key)
		if err != nil {
			logger.Error("Failed to read tmp_body: %v", err)
			return err
		}

		hash, err := html.GenerateHTMLHash(string(body))
		if err != nil {
			logger.Error("Failed to generate structural hash: %v", err)
		}

		if hash != "" {
			existingRecord, _ := app.FindFirstRecordByFilter(
				"endpoints",
				"hash = {:hash}",
				dbx.Params{"hash": hash},
			)
			if existingRecord != nil {
				return nil
			}
		}

		endpointsCollection, err := app.FindCollectionByNameOrId("endpoints")
		if err != nil {
			logger.Error("Failed to find endpoints collection: %v", err)
			return err
		}
		record := core.NewRecord(endpointsCollection)
		record.Set("url", e.Record.GetString("url"))
		record.Set("hash", hash)
		record.Set("query_string", e.Record.GetString("query_string"))
		record.Set("request_headers", e.Record.GetString("request_headers"))
		record.Set("extraction_status", "pending")
		record.Set("prettify_status", "pending")
		record.Set("created_at", time.Now())

		err = app.Save(record)

		if err != nil {
			logger.Error("Failed to save record: %v", err)
			return err
		}

		app.Delete(e.Record)
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess("endpoints").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("extraction_status") == "pending" {
			e.Record.Set("extraction_status", "processing")
			app.Save(e.Record)

			if err := extraction.AddExtractionJob(app, e.Record); err != nil {
				logger.Error("Failed to add endpoint to extraction queue: %v", err)
				return err
			}
		}

		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("endpoints").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("prettify_status") == "pending" && e.Record.GetString("extraction_status") == "processed" {
			filePath, err := storage.GetHTMLFilePath(e.Record.GetString("url"), e.Record.GetString("hash"))
			if err != nil {
				logger.Error("Failed to get HTML file path: %v", err)
			} else {
				e.Record.Set("prettify_status", "processing")
				app.Save(e.Record)
				if err := prettify.AddPrettifyJob(app, e.Record, filePath, "html"); err != nil {
					logger.Error("Failed to add HTML to prettify queue: %v", err)
				}
			}
			mobileHash := e.Record.GetString("mobile_hash")
			if mobileHash != "" {
				mobileFilePath, err := storage.GetHTMLFilePath(e.Record.GetString("url"), e.Record.GetString("mobile_hash"))
				if err != nil {
					logger.Error("Failed to get mobile HTML file path: %v", err)
				} else {
					e.Record.Set("prettify_status", "processing")
					app.Save(e.Record)
					if err := prettify.AddPrettifyJob(app, e.Record, mobileFilePath, "html"); err != nil {
						logger.Error("Failed to add mobile HTML to prettify queue: %v", err)
					}
				}
			}
		}

		return e.Next()
	})

	// =============================================================================
	// JS_FILES HOOKS
	// =============================================================================

	app.OnRecordAfterCreateSuccess("js_files").BindFunc(func(e *core.RecordEvent) error {
		e.Record.Set("prettify_status", "pending")
		e.Record.Set("analysis_status", "pending")
		e.Record.Set("sourcemap_status", "pending")
		e.Record.Set("dechunker_status", "pending")

		fileType := e.Record.GetString("type")
		if fileType == "inline" || fileType == "chunk" {
			e.Record.Set("dechunker_status", "processed") // Skip dechunking for inline/chunk files
		}

		e.Record.Set("created_at", time.Now())
		filePath, err := storage.GetJSFilePath(e.Record.GetString("url"), e.Record.GetString("hash"))
		if err != nil {
			return err
		}
		e.Record.Set("prettify_status", "processing")
		e.Record.Set("sourcemap_status", "processing")

		app.Save(e.Record)

		prettify.AddPrettifyJob(app, e.Record, filePath, "js")
		sourcemap.AddSourcemapJob(app, e.Record)

		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("js_files").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("prettify_status") == "processed" {
			if e.Record.GetString("analysis_status") == "pending" {
				e.Record.Set("analysis_status", "processing")
				app.Save(e.Record)
				analysis.AddAnalysisJob(app, e.Record)
			}
			if e.Record.GetString("dechunker_status") == "pending" {
				e.Record.Set("dechunker_status", "processing")
				app.Save(e.Record)
				if err := dechunker.AddDechunkerJob(app, e.Record); err != nil {
					logger.Error("Failed to add dechunker job for %s: %v", e.Record.GetString("url"), err)
				}
			}
		}

		return e.Next()
	})

	return nil
}
