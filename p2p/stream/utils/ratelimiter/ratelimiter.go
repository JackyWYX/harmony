package ratelimiter

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"go.uber.org/ratelimit"
)

// RateLimiter is the interface to limit the incoming request.
// The purpose of rate limiter is to prevent the node from running out of resource
// for consensus on DDoS attacks.
// TODO: research, test and implement the better rate limiter algorithm
type RateLimiter interface {
	LimitRequest(stid sttypes.StreamID, request *message.Request)
}

// rateLimiter is the implementation of RateLimiter which only blocks for the global
// level
// TODO: make request weighted in rate limiter
type rateLimiter struct {
	globalLimiter ratelimit.Limiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() RateLimiter {
	return &rateLimiter{
		globalLimiter: ratelimit.New(100),
	}
}

func (rl *rateLimiter) LimitRequest(stid sttypes.StreamID, request *message.Request) {
	rl.globalLimiter.Take()
}
