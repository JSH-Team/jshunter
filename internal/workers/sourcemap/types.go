package sourcemap

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SourcemapJob represents a job for processing sourcemaps
type SourcemapJob struct {
	App    *pocketbase.PocketBase
	Record *core.Record
}

// SourcemapWorkerPool manages a pool of workers for sourcemap processing
type SourcemapWorkerPool struct {
	workers   int
	jobQueue  chan SourcemapJob
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}
