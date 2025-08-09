package dechunker

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	// Global dechunker worker pool instance
	globalDechunkerPool *DechunkerWorkerPool
)

// SetGlobalDechunkerPool sets the global dechunker worker pool instance
func SetGlobalDechunkerPool(pool *DechunkerWorkerPool) {
	globalDechunkerPool = pool
}

// AddDechunkerJob adds a dechunker job to the global queue
func AddDechunkerJob(app *pocketbase.PocketBase, jsFileRecord *core.Record) error {
	if globalDechunkerPool == nil {
		return fmt.Errorf("dechunker worker pool not initialized")
	}

	if !globalDechunkerPool.IsRunning() {
		return fmt.Errorf("dechunker worker pool is not running")
	}

	job := DechunkerJob{
		App:    app,
		Record: jsFileRecord,
	}

	if err := globalDechunkerPool.SubmitJob(job); err != nil {
		return fmt.Errorf("failed to submit dechunker job: %w", err)
	}

	// Silently add to queue - only log completion
	return nil
}

// NewDechunkerWorkerPool creates a new dechunker worker pool
func NewDechunkerWorkerPool(maxWorkers int, queueSize int) *DechunkerWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &DechunkerWorkerPool{
		workers:   maxWorkers,
		jobQueue:  make(chan DechunkerJob, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}
}

// Start initializes and starts the dechunker worker pool
func (p *DechunkerWorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("dechunker worker pool is already running")
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the dechunker worker pool
func (p *DechunkerWorkerPool) Stop() error {
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

// SubmitJob submits a dechunker job to the pool
func (p *DechunkerWorkerPool) SubmitJob(job DechunkerJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return fmt.Errorf("dechunker worker pool is not running")
	}

	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("dechunker worker pool is shutting down")
	default:
		return fmt.Errorf("dechunker job queue is full")
	}
}

// GetQueueSize returns the current number of jobs in the queue
func (p *DechunkerWorkerPool) GetQueueSize() int {
	return len(p.jobQueue)
}

// IsRunning returns whether the worker pool is currently running
func (p *DechunkerWorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// worker is the main worker function that processes dechunker jobs
func (p *DechunkerWorkerPool) worker(workerID int) {
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
