package downloader

import (
	"context"

	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/protocols/sync"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

type syncProtocol interface {
	GetBlocksByNumber(ctx context.Context, bns []uint64, opts ...sync.Option) ([]*types.Block, sttypes.StreamID, error)
	RemoveStream(stID sttypes.StreamID) error // If a stream delivers invalid data, remove the stream
}

type blockChain interface {
	CurrentBlock() *types.Block
	InsertChain(chain types.Blocks, verifyHeaders bool) (int, error)
	ShardID() uint32
}
