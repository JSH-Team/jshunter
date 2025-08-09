package prettify

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// PrettifyJob represents a job for prettifying content
type PrettifyJob struct {
	Record   *core.Record
	Content  string
	FilePath string
	Type     string
	Context  context.Context
	App      *pocketbase.PocketBase
}

// PrettifyWorkerPool manages a pool of workers for prettifying content
type PrettifyWorkerPool struct {
	workers   int
	jobQueue  chan PrettifyJob
	workerWg  sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}
