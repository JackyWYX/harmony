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
	String() string

	GetRequestMessage() *message.Request
	ValidateResponse(resp *message.Response) error
}
