package sync

import (
	"fmt"
	"math/big"

	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/core/types"
	shardingconfig "github.com/harmony-one/harmony/internal/configs/sharding"
	"github.com/pkg/errors"
)

// chainHelper is the adapter for blockchain which is friendlier to unit test.
type chainHelper interface {
	getEpochState(epoch uint64) (*EpochStateResult, error)
	getBlocks(bns []uint64) []*types.Block
}

type chainHelperImpl struct {
	chain    engine.ChainReader
	schedule shardingconfig.Schedule
}

func newChainHelper(chain engine.ChainReader, schedule shardingconfig.Schedule) *chainHelperImpl {
	return &chainHelperImpl{
		chain:    chain,
		schedule: schedule,
	}
}

func (ch *chainHelperImpl) getBlocks(bns []uint64) []*types.Block {
	blocks := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		var block *types.Block
		header := ch.chain.GetHeaderByNumber(bn)
		if header != nil {
			block = ch.chain.GetBlock(header.Hash(), header.Number().Uint64())
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func (ch *chainHelperImpl) getEpochState(epoch uint64) (*EpochStateResult, error) {
	if ch.chain.ShardID() != 0 {
		return nil, errors.New("get epoch state currently unavailable on side chain.")
	}
	if epoch == 0 {
		return nil, errors.New("nil shard state for epoch 0")
	}
	res := &EpochStateResult{}

	targetBN := ch.schedule.EpochLastBlock(epoch - 1)
	res.Header = ch.chain.GetHeaderByNumber(targetBN)
	if res.Header == nil {
		// we still don't have the given epoch
		return res, nil
	}
	epochBI := new(big.Int).SetUint64(epoch)
	if ch.chain.Config().IsPreStaking(epochBI) {
		// For epoch before preStaking, only hash is stored in header
		ss, err := ch.chain.ReadShardState(epochBI)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			return nil, fmt.Errorf("missing shard state for [EPOCH-%v]", epoch)
		}
		res.State = ss
	}
	return res, nil
}
