package core

import (
	"errors"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/harmony-one/harmony/block"
	consensus_engine "github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/internal/params"
	"github.com/harmony-one/harmony/shard"
	lru "github.com/hashicorp/golang-lru"
)

// EpochChain is the structure which aims to preserve the epoch transition information.
// It is responsible for read / write of state shard and verification on the epoch blocks.
// The epoch blocks are defined as the last the block of the last epoch since the block
// contains the state shard of the next epoch.
// TODO: make the EpochChain inside HeaderChain
type EpochChain struct {
	config *params.ChainConfig

	chainDb ethdb.Database

	curShardState *shard.State

	shardStateCache *lru.Cache
	epochBlockCache *lru.Cache

	engine consensus_engine.Engine
}

func (ec *EpochChain) InsertHeader(header *block.Header) error {
	if header == nil {
		return errors.New("nil header")
	}
	if !header.IsLastBlockInEpoch() {
		// not an epoch block or not staking epoch, skipping.
		return nil
	}

}
