package prettify

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	// Global prettify worker pool instance
	globalPrettifyPool *PrettifyWorkerPool
)

// SetGlobalPrettifyPool sets the global prettify worker pool instance
func SetGlobalPrettifyPool(pool *PrettifyWorkerPool) {
	globalPrettifyPool = pool
}

// AddPrettifyJob adds a prettify job to the global queue
func AddPrettifyJob(app *pocketbase.PocketBase, record *core.Record, filePath string, fileType string) error {
	if globalPrettifyPool == nil {
		return fmt.Errorf("prettify worker pool not initialized")
	}

	if !globalPrettifyPool.IsRunning() {
		return fmt.Errorf("prettify worker pool is not running")
	}

	job := PrettifyJob{
		Record:   record,
		FilePath: filePath,
		Context:  context.Background(),
		App:      app,
		Type:     fileType,
	}

	if err := globalPrettifyPool.SubmitJob(job); err != nil {
		return fmt.Errorf("failed to submit prettify job: %w", err)
	}

	return nil
}

// NewPrettifyWorkerPool creates a new prettify worker pool
func NewPrettifyWorkerPool(maxWorkers int, queueSize int) *PrettifyWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &PrettifyWorkerPool{
		workers:   maxWorkers,
		jobQueue:  make(chan PrettifyJob, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}
}

// Start initializes and starts the prettify worker pool
func (p *PrettifyWorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("prettify worker pool is already running")
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the prettify worker pool
func (p *PrettifyWorkerPool) Stop() error {
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

// SubmitJob submits a prettify job to the pool
func (p *PrettifyWorkerPool) SubmitJob(job PrettifyJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return fmt.Errorf("prettify worker pool is not running")
	}

	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("prettify worker pool is shutting down")
	default:
		return fmt.Errorf("prettify job queue is full")
	}
}

// IsRunning returns whether the worker pool is currently running
func (p *PrettifyWorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// worker is the main worker function that processes prettify jobs
func (p *PrettifyWorkerPool) worker(workerID int) {
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
