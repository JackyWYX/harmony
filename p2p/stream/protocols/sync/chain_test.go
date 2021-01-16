package sync

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	protobuf "github.com/golang/protobuf/proto"
	"github.com/harmony-one/harmony/block"
	"github.com/harmony-one/harmony/core/types"
	syncpb "github.com/harmony-one/harmony/p2p/stream/protocols/sync/message"
	"github.com/harmony-one/harmony/shard"
)

type testChainHelper struct{}

func (tch *testChainHelper) getBlocks(bns []uint64) []*types.Block {
	blocks := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		header := testHeader.Copy()
		header.SetNumber(big.NewInt(int64(bn)))
		blocks = append(blocks, types.NewBlockWithHeader(&block.Header{Header: header}))
	}
	return blocks
}

func (tch *testChainHelper) getEpochState(epoch uint64) (*EpochStateResult, error) {
	header := &block.Header{Header: testHeader.Copy()}
	header.SetEpoch(big.NewInt(int64(epoch - 1)))

	state := testEpochState.DeepCopy()
	state.Epoch = big.NewInt(int64(epoch))

	return &EpochStateResult{
		Header: header,
		State:  state,
	}, nil
}

func checkBlocksResult(bns []uint64, b []byte) error {
	var msg = &syncpb.Message{}
	if err := protobuf.Unmarshal(b, msg); err != nil {
		return err
	}
	gbResp, err := msg.GetBlocksByNumberResponse()
	if err != nil {
		return err
	}
	if len(gbResp.BlocksBytes) == 0 {
		return errors.New("nil response from GetBlocksByNumber")
	}
	blocks, err := decodeBlocksBytes(gbResp.BlocksBytes)
	if err != nil {
		return err
	}
	if len(blocks) != len(bns) {
		return errors.New("unexpected blocks number")
	}
	for i, bn := range bns {
		blk := blocks[i]
		if bn != blk.NumberU64() {
			return errors.New("unexpected number of a block")
		}
	}
	return nil
}

func decodeBlocksBytes(bbs [][]byte) ([]*types.Block, error) {
	blocks := make([]*types.Block, 0, len(bbs))

	for _, bb := range bbs {
		var block *types.Block
		if err := rlp.DecodeBytes(bb, &block); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func checkEpochStateResult(epoch uint64, b []byte) error {
	var msg = &syncpb.Message{}
	if err := protobuf.Unmarshal(b, msg); err != nil {
		return err
	}
	geResp, err := msg.GetEpochStateResponse()
	if err != nil {
		return err
	}
	var (
		header     *block.Header
		epochState *shard.State
	)
	if err := rlp.DecodeBytes(geResp.HeaderBytes, &header); err != nil {
		return err
	}
	if err := rlp.DecodeBytes(geResp.ShardState, &epochState); err != nil {
		return err
	}
	if header.Epoch().Uint64() != epoch-1 {
		return fmt.Errorf("unexpected epoch of header %v / %v", header.Epoch(), epoch-1)
	}
	if epochState.Epoch.Uint64() != epoch {
		return fmt.Errorf("unexpected epoch of shard state %v / %v", epochState.Epoch.Uint64(), epoch)
	}
	return nil
}
