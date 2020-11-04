package streammanager

import "time"

const (
	defHardLoCap = 4  // discovery trigger immediately when size smaller than this number
	defSoftLoCap = 16 // discovery trigger for routine check
	defHiCap     = 64 // Hard cap of the stream number
	defDiscBatch = 16 // batch size for discovery
)

const (
	checkInterval  = 5 * time.Minute // interval for checking peer number
	discTimeout    = 60 * time.Second
	connectTimeout = 60 * time.Second
)

var defConfig = Config{
	HardLoCap: defHardLoCap,
	SoftLoCap: defSoftLoCap,
	HiCap:     defHiCap,
	DiscBatch: defDiscBatch,
}

// Config is the config for stream manager
type Config struct {
	// HardLoCap is low cap of stream number that immediately trigger discovery
	HardLoCap int
	// SoftLoCap is low cap of stream number that will trigger discovery during stream check
	SoftLoCap int
	// HiCap is the high cap of stream number
	HiCap int
	// DiscBatch is the size of each discovery
	DiscBatch int
}

// Option is the function to modify the config of stream manager
type Option func(*Config)

// WithHardLoCap set HardLoCap to val
func WithHardLoCap(val int) Option {
	return func(c *Config) {
		c.HardLoCap = val
	}
}

// WithSoftLoCap set SoftLoCap to val
func WithSoftLoCap(val int) Option {
	return func(c *Config) {
		c.SoftLoCap = val
	}
}

// WithHiCap set HighCap to val
func WithHiCap(val int) Option {
	return func(c *Config) {
		c.HiCap = val
	}
}

// WithDiscBatch set discover batch size to val
func WithDiscBatch(val int) Option {
	return func(c *Config) {
		c.DiscBatch = val
	}
}

func (c *Config) applyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
