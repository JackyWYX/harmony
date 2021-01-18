package downloader

const (
	defaultConcurrency      int = 8
	defaultBlocksPerRequest int = 5

	// hard cap of max number of blocks in resultQueue
	queueMaxSize int = 200

	// soft cap of size in resultQueue. When the queue size is larger than this limit,
	// no more request will be assigned to workers to wait for InsertChain to finish.
	softQueueCap int = 100
)

type Config struct {
	Concurrency      *int // number of workers doing requests
	BlocksPerRequest *int
}

func (c *Config) applyDefaults() {
	if c.Concurrency == nil || *c.Concurrency == 0 {
		val := defaultConcurrency
		c.Concurrency = &val
	}
	if c.BlocksPerRequest == nil || *c.BlocksPerRequest == 0 {
		val := defaultBlocksPerRequest
		c.BlocksPerRequest = &val
	}
}
