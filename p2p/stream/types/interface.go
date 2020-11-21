package sttypes

import (
	"github.com/harmony-one/harmony/p2p/stream/message"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	"github.com/hashicorp/go-version"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Protocol is the interface of protocol to be registered to libp2p.
type Protocol interface {
	p2ptypes.LifeCycle

	Specifier() string
	Version() *version.Version
	ProtoID() ProtoID
	Match(string) bool
	HandleStream(st libp2p_network.Stream)
}

// Request is the interface of a single request.
type Request interface {
	ReqID() uint64
	SetReqID(rid uint64)
	String() string

	IsSupportedByProto(ProtoSpec) bool
	GetRequestMessage() *message.Request
}
