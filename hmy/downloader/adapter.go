package downloader

import (
	"context"

	"github.com/harmony-one/harmony/core/types"
	syncproto "github.com/harmony-one/harmony/p2p/stream/protocols/sync"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

type syncProtocol interface {
	GetCurrentBlockNumber(ctx context.Context, opts ...syncproto.Option) (uint64, sttypes.StreamID, error)
	GetBlocksByNumber(ctx context.Context, bns []uint64, opts ...syncproto.Option) ([]*types.Block, sttypes.StreamID, error)

	RemoveStream(stID sttypes.StreamID) error // If a stream delivers invalid data, remove the stream
	NumStreams() int
}

type blockChain interface {
	CurrentBlock() *types.Block
	InsertChain(chain types.Blocks, verifyHeaders bool) (int, error)
	ShardID() uint32
}
