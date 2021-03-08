package downloader

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/internal/params"
	"github.com/harmony-one/harmony/p2p/stream/common/streammanager"
	syncproto "github.com/harmony-one/harmony/p2p/stream/protocols/sync"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/harmony-one/harmony/shard"
)

type syncProtocol interface {
	GetCurrentBlockNumber(ctx context.Context, opts ...syncproto.Option) (uint64, sttypes.StreamID, error)
	GetBlocksByNumber(ctx context.Context, bns []uint64, opts ...syncproto.Option) ([]*types.Block, sttypes.StreamID, error)
	GetBlockHashes(ctx context.Context, bns []uint64, opts ...syncproto.Option) ([]common.Hash, sttypes.StreamID, error)
	GetBlocksByHashes(ctx context.Context, hs []common.Hash, opts ...syncproto.Option) ([]*types.Block, sttypes.StreamID, error)

	RemoveStream(stID sttypes.StreamID) // If a stream delivers invalid data, remove the stream
	SubscribeAddStreamEvent(ch chan<- streammanager.EvtStreamAdded) event.Subscription
	NumStreams() int
}

type blockChain interface {
	CurrentBlock() *types.Block
	InsertChain(chain types.Blocks, verifyHeaders bool) (int, error)
	ShardID() uint32

	// Fields used for consensus helper to verify block signature
	ReadShardState(epoch *big.Int) (*shard.State, error)
	Config() *params.ChainConfig
	WriteCommitSig(blockNum uint64, lastCommits []byte) error
}

// consensusHelper is the interface to do extra step of block signatures
// (block.commitSigAndBitmap)
type consensusHelper interface {
	verifyBlockSignature(block *types.Block) error
	writeBlockSignature(block *types.Block) error
}
