package sttypes

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
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
	LimitRequest(stream Stream, request message.Request)
}

// StreamManager is adapter interface to handle new stream and closed stream events
type StreamManager interface {
	HandleNewStream(stream Stream) error
	HandleStreamErr(stream Stream, err error)
}

// Stream is the interface for streams for each service.
type Stream interface {
	PeerID() p2ptypes.PeerID
	ProtoID() ProtoID
	ProtoSpec() ProtoSpec
	Direction() libp2p_network.Direction

	Close()
}
