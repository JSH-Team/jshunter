package dechunker

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// DechunkerJob represents a job for extracting chunks from JavaScript files
type DechunkerJob struct {
	App    *pocketbase.PocketBase
	Record *core.Record
}

// DechunkerWorkerPool manages a pool of workers for JavaScript chunk extraction
type DechunkerWorkerPool struct {
	workers   int
	jobQueue  chan DechunkerJob
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}

// ChunkURL represents a discovered chunk URL
type ChunkURL struct {
	URL      string                 // The chunk URL
	Type     string                 // "webpack", "vite", or "unknown"
	ChunkID  string                 // The chunk identifier (if available)
	Metadata map[string]interface{} // Additional metadata
}
