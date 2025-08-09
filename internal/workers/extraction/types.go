package extraction

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ExtractionJob represents a job for extracting content from endpoints
type ExtractionJob struct {
	App     *pocketbase.PocketBase
	Record  *core.Record
	Context context.Context
}

// ExtractionWorkerPool manages a pool of workers for content extraction
type ExtractionWorkerPool struct {
	workers   int
	jobQueue  chan ExtractionJob
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}

// JSFileResult represents a JavaScript file extracted from an endpoint
type JSFileResult struct {
	URL     string
	Content string
	Type    string // "normal", "inline", "mobile"
}
