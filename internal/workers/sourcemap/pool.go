package sourcemap

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Global sourcemap pool instance
var globalSourcemapPool *SourcemapWorkerPool

// SetGlobalSourcemapPool sets the global sourcemap worker pool instance
func SetGlobalSourcemapPool(pool *SourcemapWorkerPool) {
	globalSourcemapPool = pool
}

// AddSourcemapJob adds a sourcemap job to the global pool
func AddSourcemapJob(app *pocketbase.PocketBase, jsFileRecord *core.Record) error {
	if globalSourcemapPool == nil {
		return fmt.Errorf("sourcemap worker pool not initialized")
	}

	job := SourcemapJob{
		App:    app,
		Record: jsFileRecord,
	}

	select {
	case globalSourcemapPool.jobQueue <- job:
		return nil
	default:
		return fmt.Errorf("sourcemap queue is full")
	}
}

// NewSourcemapWorkerPool creates a new sourcemap worker pool
func NewSourcemapWorkerPool(maxWorkers int, queueSize int) *SourcemapWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &SourcemapWorkerPool{
		workers:   maxWorkers,
		jobQueue:  make(chan SourcemapJob, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}
}

// Start initializes and starts the sourcemap worker pool
func (p *SourcemapWorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("sourcemap worker pool is already running")
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the sourcemap worker pool
func (p *SourcemapWorkerPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	// Cancel context to signal workers to stop
	p.cancel()

	// Close job queue to prevent new jobs
	close(p.jobQueue)

	// Wait for all workers to finish
	p.workerWg.Wait()

	p.isRunning = false
	return nil
}

// IsRunning returns whether the worker pool is currently running
func (p *SourcemapWorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// worker is the main worker function that processes sourcemap jobs
func (p *SourcemapWorkerPool) worker(workerID int) {
	defer p.workerWg.Done()

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				return
			}

			// Process the job
			p.processJob(workerID, job)

		case <-p.ctx.Done():
			return
		}
	}
}
