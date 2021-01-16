package requestmanager

import sttypes "github.com/harmony-one/harmony/p2p/stream/types"

// Request is the additional instruction for requests.
// Currently, two options are supported:
// 1. WithHighPriority
// 2. WithBlacklist
type RequestOption func(*request)

// WithHighPriority is the request option to do request with higher priority.
// High priority requests are done first.
func WithHighPriority() RequestOption {
	return func(req *request) {
		req.priority = reqPriorityHigh
	}
}

// WithBlacklist is the request option not to assign the request to the blacklisted
// stream ID.
func WithBlacklist(blacklist []sttypes.StreamID) RequestOption {
	return func(req *request) {
		for _, stid := range blacklist {
			req.addBlacklistedStream(stid)
		}
	}
}
