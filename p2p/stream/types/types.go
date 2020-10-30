package sttypes

import (
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Metadata contains the necessary information for stream management
type Metadata struct {
	PeerID    p2ptypes.PeerID
	ProtoID   ProtoID
	ProtoSpec ProtoSpec
	Direction Direction
}

// Direction is the direction of libp2p stream
type Direction libp2p_network.Direction

const (
	// DirInbound defines a stream as an inbound connection
	DirInbound = libp2p_network.DirInbound

	// DirInbound defines a stream as an outbound connection
	DirOutbound = libp2p_network.DirOutbound
)
