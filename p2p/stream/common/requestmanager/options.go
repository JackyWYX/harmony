package requestmanager

import sttypes "github.com/harmony-one/harmony/p2p/stream/types"

type RequestOption func(*request)

// WithHighPriority is the request option to do request with higher priority.
// High priority requests are done first.
func WithHighPriority() RequestOption {
	return func(req *request) {
		req.priority = reqPriorityHigh
	}
}

// WithBlacklist is the request option to do request without the given stream
func WithBlacklist(blacklist []sttypes.StreamID) RequestOption {
	return func(req *request) {
		req.blacklist = append(req.blacklist, blacklist...)
	}
}
