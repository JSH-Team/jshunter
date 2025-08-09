package analysis

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	// Global analysis worker pool instance
	globalAnalysisPool *AnalysisWorkerPool
)

// SetGlobalAnalysisPool sets the global analysis worker pool instance
func SetGlobalAnalysisPool(pool *AnalysisWorkerPool) {
	globalAnalysisPool = pool
}

// AddAnalysisJob adds an analysis job to the global queue
func AddAnalysisJob(app *pocketbase.PocketBase, jsFileRecord *core.Record) error {
	if globalAnalysisPool == nil {
		return fmt.Errorf("analysis worker pool not initialized")
	}

	if !globalAnalysisPool.IsRunning() {
		return fmt.Errorf("analysis worker pool is not running")
	}

	job := AnalysisJob{
		App:    app,
		Record: jsFileRecord,
	}

	if err := globalAnalysisPool.SubmitJob(job); err != nil {
		return fmt.Errorf("failed to submit analysis job: %w", err)
	}

	// Silently add to queue - only log completion
	return nil
}

// NewAnalysisWorkerPool creates a new analysis worker pool
func NewAnalysisWorkerPool(maxWorkers int, queueSize int) *AnalysisWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &AnalysisWorkerPool{
		workers:   maxWorkers,
		jobQueue:  make(chan AnalysisJob, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}
}

// Start initializes and starts the analysis worker pool
func (p *AnalysisWorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("analysis worker pool is already running")
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the analysis worker pool
func (p *AnalysisWorkerPool) Stop() error {
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

// SubmitJob submits an analysis job to the pool
func (p *AnalysisWorkerPool) SubmitJob(job AnalysisJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return fmt.Errorf("analysis worker pool is not running")
	}

	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("analysis worker pool is shutting down")
	default:
		return fmt.Errorf("analysis job queue is full")
	}
}

// GetQueueSize returns the current number of jobs in the queue
func (p *AnalysisWorkerPool) GetQueueSize() int {
	return len(p.jobQueue)
}

// IsRunning returns whether the worker pool is currently running
func (p *AnalysisWorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// worker is the main worker function that processes analysis jobs
func (p *AnalysisWorkerPool) worker(workerID int) {
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
