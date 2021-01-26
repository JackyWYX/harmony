package downloader

const (
	blocksPerRequest int = 10 // number of blocks for each request
	blocksPerInsert  int = 50 // number of blocks for each insert batch

	lastMileThres uint64 = 50

	// soft cap of size in resultQueue. When the queue size is larger than this limit,
	// no more request will be assigned to workers to wait for InsertChain to finish.
	softQueueCap int = 100
)

// Config is the downloader config
type Config struct {
	Concurrency int // Number of concurrent sync requests
	MinStreams  int // Minimum number of streams to do sync
}
