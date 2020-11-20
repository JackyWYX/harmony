package requestmanager

import (
	"context"

	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
)

// Requester is the interface to do request
type Requester interface {
	DoRequest(ctx context.Context, request sttypes.Request) (*message.Response, sttypes.StreamID, error)
}

// Deliverer is the interface to deliver a response
type Deliverer interface {
	DeliverResponse(stID sttypes.StreamID, resp *message.Response)
}

// RequestManager manages over the requests
type RequestManager interface {
	p2ptypes.LifeCycle
	Requester
	Deliverer
}
