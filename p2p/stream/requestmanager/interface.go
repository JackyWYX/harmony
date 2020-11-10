package requestmanager

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

// Requester is the interface to do request
type Requester interface {
	DoRequest(request sttypes.Request) (<-chan *message.Response, error)
}

// Deliverer is the interface to deliver a response
type Deliverer interface {
	DeliverResponse(response *message.Response)
}

// RequestManager manages over the requests
type RequestManager interface {
	Requester
	Deliverer
}

// streamManager is the adapter interface for stream manager which supports stream event
// notification.
type streamManager interface {
}
