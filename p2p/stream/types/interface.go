package sttypes

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	"github.com/hashicorp/go-version"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Protocol is the interface of protocol to be registered to libp2p.
type Protocol interface {
	Version() *version.Version
	ProtoID() ProtoID
	Match(string) bool
	HandleStream(st libp2p_network.Stream)
}

// RateLimiter is the adapter interface to limit the incoming request
type RateLimiter interface {
	LimitRequest(stream Stream, request *message.Request)
}

// Request is the interface of a single request.
type Request interface {
	ReqID() uint64
	SetReqID(rid uint64)

	ValidateResponse(resp *message.Response) error
	EstimateCost() uint64 // TODO: Implement this to load balance the requests
}

// Requester is the interface to do request
type Requester interface {
	DoRequest(request Request) (<-chan *message.Response, error)
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
