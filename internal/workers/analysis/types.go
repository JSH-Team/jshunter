package analysis

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// AnalysisJob represents a job for analyzing JavaScript content
type AnalysisJob struct {
	App    *pocketbase.PocketBase
	Record *core.Record
}

// AnalysisWorkerPool manages a pool of workers for JavaScript analysis
type AnalysisWorkerPool struct {
	workers   int
	jobQueue  chan AnalysisJob
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}
