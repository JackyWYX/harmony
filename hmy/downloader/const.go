package downloader

const (
	concurrency      int = 8
	blocksPerRequest int = 10 // number of blocks for each request
	blocksPerBatch   int = 10 // number of blocks for each insert batch

	// soft cap of size in resultQueue. When the queue size is larger than this limit,
	// no more request will be assigned to workers to wait for InsertChain to finish.
	softQueueCap int = 100
)
