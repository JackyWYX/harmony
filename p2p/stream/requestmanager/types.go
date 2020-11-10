package requestmanager

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

// stream is the wrapped version of sttypes.Stream.
type stream struct {
	sttypes.Stream
	pending *request // request that is currently processing
}

// request is the wrapped request
type request struct {
	sttypes.Request
	owner *stream

	resp   chan *message.Response
	cancel func()
}
