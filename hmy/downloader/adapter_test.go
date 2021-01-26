package downloader

import (
	"context"
	"fmt"
	"sync"

	"github.com/harmony-one/harmony/core/types"
	syncproto "github.com/harmony-one/harmony/p2p/stream/protocols/sync"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

type testBlockChain struct {
	curBN uint64
	lock  sync.Mutex
}

func newTestBlockChain(curBN uint64) *testBlockChain {
	return &testBlockChain{
		curBN: curBN,
	}
}

func (bc *testBlockChain) CurrentBlock() *types.Block {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	return makeTestBlock(bc.curBN)
}

func (bc *testBlockChain) currentBlockNumber() uint64 {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.curBN
}

func (bc *testBlockChain) InsertChain(chain types.Blocks, verifyHeaders bool) (int, error) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for i, block := range chain {
		if block.NumberU64() <= bc.curBN {
			continue
		}
		if block.NumberU64() != bc.curBN+1 {
			return i, fmt.Errorf("not expected block number: %v / %v", block.NumberU64(), bc.curBN+1)
		}
		bc.curBN++
	}
	return len(chain), nil
}

func (bc *testBlockChain) ShardID() uint32 {
	return 0
}

type testSyncProtocol struct {
	streamIDs   []sttypes.StreamID
	remoteChain *testBlockChain

	curIndex int
	lock     sync.Mutex
}

func newTestSyncProtocol(targetBN uint64) *testSyncProtocol {
	return &testSyncProtocol{
		streamIDs:   makeStreamIDs(32),
		remoteChain: newTestBlockChain(targetBN),
		curIndex:    0,
	}
}

func (sp *testSyncProtocol) GetCurrentBlockNumber(ctx context.Context, opts ...syncproto.Option) (uint64, sttypes.StreamID, error) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	bn := sp.remoteChain.currentBlockNumber()

	return bn, sp.nextStreamID(), nil
}

func (sp *testSyncProtocol) GetBlocksByNumber(ctx context.Context, bns []uint64, opts ...syncproto.Option) ([]*types.Block, sttypes.StreamID, error) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	res := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		if bn > sp.remoteChain.currentBlockNumber() {
			res = append(res, nil)
		} else {
			res = append(res, makeTestBlock(bn))
		}
	}
	return res, sp.nextStreamID(), nil
}

func (sp *testSyncProtocol) RemoveStream(target sttypes.StreamID) error {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	for i, stid := range sp.streamIDs {
		if stid == target {
			if i == len(sp.streamIDs)-1 {
				sp.streamIDs = sp.streamIDs[:i]
			} else {
				sp.streamIDs = append(sp.streamIDs[:i], sp.streamIDs[i+1:]...)
			}
		}
	}
	return nil
}

func (sp *testSyncProtocol) NumStreams() int {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	return len(sp.streamIDs)
}

func (sp *testSyncProtocol) nextStreamID() sttypes.StreamID {
	if sp.curIndex >= len(sp.streamIDs) {
		sp.curIndex = 0
	}
	index := sp.curIndex
	sp.curIndex++
	if sp.curIndex >= len(sp.streamIDs) {
		sp.curIndex = 0
	}
	return sp.streamIDs[index]
}

func makeStreamIDs(size int) []sttypes.StreamID {
	res := make([]sttypes.StreamID, 0, size)
	for i := 0; i != size; i++ {
		res = append(res, makeStreamID(i))
	}
	return res
}

func makeStreamID(index int) sttypes.StreamID {
	return sttypes.StreamID(fmt.Sprintf("test stream %v", index))
}
