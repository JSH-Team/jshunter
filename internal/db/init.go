package db

import (
	"fmt"
	"jshunter/internal/config"
	"jshunter/internal/utils/logger"
	"jshunter/internal/workers/analysis"
	"jshunter/internal/workers/dechunker"
	"jshunter/internal/workers/extraction"
	"jshunter/internal/workers/prettify"
	"jshunter/internal/workers/sourcemap"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	extractionWorkerPool *extraction.ExtractionWorkerPool
	prettifyWorkerPool   *prettify.PrettifyWorkerPool
	sourcemapWorkerPool  *sourcemap.SourcemapWorkerPool
	analysisWorkerPool   *analysis.AnalysisWorkerPool
	dechunkerWorkerPool  *dechunker.DechunkerWorkerPool
)

func RunDB() {

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:  config.GetDbPath(),
		HideStartBanner: true,
	})

	// Initialize extraction worker pool
	extractionWorkerPool = extraction.NewExtractionWorkerPool(
		config.MaxConcurrentBrowsers,
		config.QueueBufferSize,
	)

	if err := extractionWorkerPool.Start(); err != nil {
		return
	}

	// Initialize prettify worker pool
	prettifyWorkerPool = prettify.NewPrettifyWorkerPool(
		config.MaxConcurrentPrettify,
		config.PrettifyQueueSize,
	)

	if err := prettifyWorkerPool.Start(); err != nil {
		return
	}

	// Initialize sourcemap worker pool
	sourcemapWorkerPool = sourcemap.NewSourcemapWorkerPool(
		config.MaxConcurrentSourcemaps,
		config.SourcemapQueueSize,
	)

	if err := sourcemapWorkerPool.Start(); err != nil {
		return
	}

	// Initialize analysis worker pool
	analysisWorkerPool = analysis.NewAnalysisWorkerPool(
		config.MaxConcurrentAnalysis,
		config.AnalysisQueueSize,
	)

	if err := analysisWorkerPool.Start(); err != nil {
		return
	}

	// Initialize dechunker worker pool
	dechunkerWorkerPool = dechunker.NewDechunkerWorkerPool(
		config.MaxConcurrentDechunker,
		config.DechunkerQueueSize,
	)

	if err := dechunkerWorkerPool.Start(); err != nil {
		return
	}

	// Set global worker pools for utility functions
	extraction.SetGlobalExtractionPool(extractionWorkerPool)
	prettify.SetGlobalPrettifyPool(prettifyWorkerPool)
	sourcemap.SetGlobalSourcemapPool(sourcemapWorkerPool)
	analysis.SetGlobalAnalysisPool(analysisWorkerPool)
	dechunker.SetGlobalDechunkerPool(dechunkerWorkerPool)

	// Register crons and hooks
	RegisterHooks(app)

	// Handle graceful shutdown
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		// Silently stop all worker pools
		if err := extractionWorkerPool.Stop(); err != nil {
			logger.Error("Error stopping extraction worker pool: %v", err)
		}

		if err := prettifyWorkerPool.Stop(); err != nil {
			logger.Error("Error stopping prettify worker pool: %v", err)
		}

		if err := sourcemapWorkerPool.Stop(); err != nil {
			logger.Error("Error stopping sourcemap worker pool: %v", err)
		}

		if err := analysisWorkerPool.Stop(); err != nil {
			logger.Error("Error stopping analysis worker pool: %v", err)
		}

		if err := dechunkerWorkerPool.Stop(); err != nil {
			logger.Error("Error stopping dechunker worker pool: %v", err)
		}

		return e.Next()
	})

	RegisterRoutes(app)

	// Hook para ejecutar después de que la base de datos esté completamente lista
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		// Ejecutar recuperación de jobs pendientes después del bootstrap
		go func() {
			// Pequeña pausa para asegurar que todo esté listo
			time.Sleep(2 * time.Second)
			if pbApp, ok := e.App.(*pocketbase.PocketBase); ok {
				recoverPendingJobs(pbApp)
			}
		}()
		return e.Next()
	})

	os.Args = []string{"pocketbase", "serve", "--http", fmt.Sprintf("localhost:%d", config.Port)}
	logger.Info("JSHunter server started on port %d", config.Port)

	if err := app.Start(); err != nil {
		logger.Error(err.Error())
	}
}

// GetWorkerPool returns the global extraction worker pool
func GetWorkerPool() *extraction.ExtractionWorkerPool {
	return extractionWorkerPool
}

// recoverPendingJobs recovers all pending jobs and queues them for processing
func recoverPendingJobs(app *pocketbase.PocketBase) {
	logger.Info("Starting recovery of pending jobs...")

	pendingEndpoints, err := app.FindRecordsByFilter(
		"endpoints",
		"extraction_status = 'pending'  || extraction_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit - process all pending
		0,
	)

	if err != nil {
		logger.Error("Error finding pending endpoints: %v", err)
	} else if len(pendingEndpoints) > 0 {
		logger.Info("Found %d pending extraction jobs to recover", len(pendingEndpoints))

		for _, record := range pendingEndpoints {
			if err := extraction.AddExtractionJob(app, record); err != nil {
				logger.Error("Failed to queue recovery extraction job for %s: %v", record.GetString("url"), err)
			}
		}
	}

	// 2. Recover pending prettify jobs (endpoints and js_files with prettify_status = 'pending')

	// 2a. Endpoints with pending prettify
	pendingEndpointPrettify, err := app.FindRecordsByFilter(
		"endpoints",
		"prettify_status = 'pending' || prettify_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit
		0,
	)

	if err != nil {
		logger.Error("Error finding pending endpoint prettify jobs: %v", err)
	} else if len(pendingEndpointPrettify) > 0 {
		logger.Info("Found %d pending endpoint prettify jobs to recover", len(pendingEndpointPrettify))

		for _, record := range pendingEndpointPrettify {
			job := prettify.PrettifyJob{
				Record: record,
				App:    app,
			}
			if err := prettifyWorkerPool.SubmitJob(job); err != nil {
				logger.Error("Failed to queue recovery prettify job for endpoint %s: %v", record.GetString("url"), err)
			}
		}
	}

	// 2b. JS files with pending prettify
	pendingJSPrettify, err := app.FindRecordsByFilter(
		"js_files",
		"prettify_status = 'pending' || prettify_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit
		0,
	)

	if err != nil {
		logger.Error("Error finding pending JS prettify jobs: %v", err)
	} else if len(pendingJSPrettify) > 0 {
		logger.Info("Found %d pending JS prettify jobs to recover", len(pendingJSPrettify))

		for _, record := range pendingJSPrettify {
			job := prettify.PrettifyJob{
				Record: record,
				App:    app,
			}
			if err := prettifyWorkerPool.SubmitJob(job); err != nil {
				logger.Error("Failed to queue recovery prettify job for JS %s: %v", record.GetString("url"), err)
			}
		}
	}

	// 3. Recover pending sourcemap jobs (js_files with sourcemap_status = 'pending')
	pendingSourcemap, err := app.FindRecordsByFilter(
		"js_files",
		"sourcemap_status = 'pending' || sourcemap_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit
		0,
	)

	if err != nil {
		logger.Error("Error finding pending sourcemap jobs: %v", err)
	} else if len(pendingSourcemap) > 0 {
		logger.Info("Found %d pending sourcemap jobs to recover", len(pendingSourcemap))

		for _, record := range pendingSourcemap {
			if err := sourcemap.AddSourcemapJob(app, record); err != nil {
				logger.Error("Failed to queue recovery sourcemap job for %s: %v", record.GetString("url"), err)
			}
		}
	}

	// 4. Recover pending analysis jobs (js_files with analysis_status = 'pending')
	pendingAnalysis, err := app.FindRecordsByFilter(
		"js_files",
		"analysis_status = 'pending' || analysis_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit
		0,
	)

	if err != nil {
		logger.Error("Error finding pending analysis jobs: %v", err)
	} else if len(pendingAnalysis) > 0 {
		logger.Info("Found %d pending analysis jobs to recover", len(pendingAnalysis))

		for _, record := range pendingAnalysis {
			job := analysis.AnalysisJob{
				Record: record,
				App:    app,
			}
			if err := analysisWorkerPool.SubmitJob(job); err != nil {
				logger.Error("Failed to queue recovery analysis job for %s: %v", record.GetString("url"), err)
			}
		}
	}

	// 5. Recover pending dechunker jobs (js_files with dechunker_status = 'pending')
	pendingDechunker, err := app.FindRecordsByFilter(
		"js_files",
		"dechunker_status = 'pending' || dechunker_status = 'processing'",
		"created_at", // Order from oldest to newest
		0,            // No limit
		0,
	)

	if err != nil {
		logger.Error("Error finding pending dechunker jobs: %v", err)
	} else if len(pendingDechunker) > 0 {
		logger.Info("Found %d pending dechunker jobs to recover", len(pendingDechunker))

		for _, record := range pendingDechunker {
			if err := dechunker.AddDechunkerJob(app, record); err != nil {
				logger.Error("Failed to queue recovery dechunker job for %s: %v", record.GetString("url"), err)
			}
		}
	}

	totalRecovered := len(pendingEndpoints) + len(pendingEndpointPrettify) + len(pendingJSPrettify) + len(pendingSourcemap) + len(pendingAnalysis) + len(pendingDechunker)
	if totalRecovered > 0 {
		logger.Info("Recovery completed: %d total pending jobs queued for processing", totalRecovered)
	} else {
		logger.Info("No pending jobs found to recover")
	}
}
