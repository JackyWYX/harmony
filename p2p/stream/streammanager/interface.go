package streammanager

import (
	"context"

	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// StreamManager handle new stream and closed stream events.
type StreamManager interface {
	Start()
	Close()

	HandleNewStream(stream sttypes.Stream) error
	RemoveStream(pid p2ptypes.PeerID) error
}

// host is the adapter interface of the libp2p host implementation.
type host interface {
	NewStream(ctx context.Context, p libp2p_peer.ID, pids ...protocol.ID) (network.Stream, error)
}
