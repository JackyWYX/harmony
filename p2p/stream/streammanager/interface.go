package streammanager

import (
	"context"

	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// StreamManager handle new stream and closed stream events.
type StreamManager interface {
	Start()
	Close()

	HandleNewStream(ctx context.Context, stream sttypes.Stream) error
	RemoveStream(ctx context.Context, stID sttypes.StreamID) error
}

// host is the adapter interface of the libp2p host implementation.
// TODO: further adapt the host
type host interface {
	ID() libp2p_peer.ID
	NewStream(ctx context.Context, p libp2p_peer.ID, pids ...protocol.ID) (network.Stream, error)
}

// peerFinder is the adapter interface of discovery.Discovery
type peerFinder interface {
	FindPeers(ctx context.Context, ns string, peerLimit int) (<-chan libp2p_peer.AddrInfo, error)
}
