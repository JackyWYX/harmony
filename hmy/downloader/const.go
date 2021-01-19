package downloader

const (
	defaultConcurrency      int = 8
	defaultBlocksPerRequest int = 10

	// soft cap of size in resultQueue. When the queue size is larger than this limit,
	// no more request will be assigned to workers to wait for InsertChain to finish.
	softQueueCap int = 100
)
