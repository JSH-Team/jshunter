package extraction

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Global extraction pool instance
var globalExtractionPool *ExtractionWorkerPool

// SetGlobalExtractionPool sets the global extraction worker pool instance
func SetGlobalExtractionPool(pool *ExtractionWorkerPool) {
	globalExtractionPool = pool
}

// AddExtractionJob adds an extraction job to the global pool
func AddExtractionJob(app *pocketbase.PocketBase, endpointRecord *core.Record) error {
	if globalExtractionPool == nil {
		return fmt.Errorf("extraction worker pool not initialized")
	}

	job := ExtractionJob{
		App:     app,
		Record:  endpointRecord,
		Context: context.Background(),
	}

	select {
	case globalExtractionPool.jobQueue <- job:
		return nil
	default:
		return fmt.Errorf("extraction queue is full")
	}
}

// AddExtractionJobs adds multiple extraction jobs to the global pool
func AddExtractionJobs(app *pocketbase.PocketBase, endpointRecords []*core.Record) error {
	if globalExtractionPool == nil {
		return fmt.Errorf("extraction worker pool not initialized")
	}

	if len(endpointRecords) == 0 {
		return nil // Nothing to add
	}

	// Check if we have enough space in the queue
	availableSpace := cap(globalExtractionPool.jobQueue) - len(globalExtractionPool.jobQueue)
	if len(endpointRecords) > availableSpace {
		return fmt.Errorf("not enough space in extraction queue: need %d slots, have %d available", len(endpointRecords), availableSpace)
	}

	// Add all jobs to the queue
	successCount := 0
	var lastError error

	for _, record := range endpointRecords {
		job := ExtractionJob{
			App:     app,
			Record:  record,
			Context: context.Background(),
		}

		select {
		case globalExtractionPool.jobQueue <- job:
			successCount++
		default:
			lastError = fmt.Errorf("extraction queue became full while adding job for %s", record.GetString("url"))
			break
		}
	}

	if lastError != nil {
		return fmt.Errorf("added %d/%d jobs before error: %w", successCount, len(endpointRecords), lastError)
	}

	return nil
}

// AddSequentialExtractionJobs adds multiple jobs to a single-worker pool for sequential processing
func AddSequentialExtractionJobs(app *pocketbase.PocketBase, endpointRecords []*core.Record) error {
	// Create a single-worker pool for sequential processing
	sequentialPool := NewExtractionWorkerPool(1, len(endpointRecords)+10)

	if err := sequentialPool.Start(); err != nil {
		return fmt.Errorf("failed to start sequential pool: %w", err)
	}

	// Add all jobs to the sequential pool
	if err := sequentialPool.SubmitJobs(app, endpointRecords); err != nil {
		sequentialPool.Stop()
		return err
	}

	// Let the pool run and clean up after all jobs are done
	go func() {
		// Wait for all jobs to complete
		for sequentialPool.GetQueueSize() > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		// Give workers time to finish current job
		time.Sleep(2 * time.Second)
		sequentialPool.Stop()
	}()

	return nil
}

// NewExtractionWorkerPool creates a new extraction worker pool
func NewExtractionWorkerPool(maxWorkers int, queueSize int) *ExtractionWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &ExtractionWorkerPool{
		workers:   maxWorkers,
		jobQueue:  make(chan ExtractionJob, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}
}

// Start initializes and starts the extraction worker pool
func (p *ExtractionWorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("extraction worker pool is already running")
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the extraction worker pool
func (p *ExtractionWorkerPool) Stop() error {
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

// GetQueueSize returns the current number of jobs in the queue
func (p *ExtractionWorkerPool) GetQueueSize() int {
	return len(p.jobQueue)
}

// GetAvailableSpace returns the available space in the queue
func (p *ExtractionWorkerPool) GetAvailableSpace() int {
	return cap(p.jobQueue) - len(p.jobQueue)
}

// SubmitJobs submits multiple jobs to the worker pool
// This method operates directly on the pool instance and includes proper synchronization.
// Example usage:
//
//	if err := pool.SubmitJobs(app, records); err != nil {
//	    log.Printf("Failed to submit batch: %v", err)
//	}
func (p *ExtractionWorkerPool) SubmitJobs(app *pocketbase.PocketBase, endpointRecords []*core.Record) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return fmt.Errorf("extraction worker pool is not running")
	}

	if len(endpointRecords) == 0 {
		return nil // Nothing to add
	}

	// Check available space
	availableSpace := p.GetAvailableSpace()
	if len(endpointRecords) > availableSpace {
		return fmt.Errorf("not enough space in extraction queue: need %d slots, have %d available", len(endpointRecords), availableSpace)
	}

	// Add all jobs
	successCount := 0
	var lastError error

	for _, record := range endpointRecords {
		job := ExtractionJob{
			App:     app,
			Record:  record,
			Context: context.Background(),
		}

		select {
		case p.jobQueue <- job:
			successCount++
		case <-p.ctx.Done():
			lastError = fmt.Errorf("worker pool is shutting down")
			break
		default:
			lastError = fmt.Errorf("extraction queue became full while adding job for %s", record.GetString("url"))
			break
		}
	}

	if lastError != nil {
		return fmt.Errorf("added %d/%d jobs before error: %w", successCount, len(endpointRecords), lastError)
	}

	return nil
}

// worker is the main worker function that processes extraction jobs
func (p *ExtractionWorkerPool) worker(workerID int) {
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
