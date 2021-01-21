package downloader

import (
	"errors"
	"sync"

	"github.com/harmony-one/harmony/core/types"
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

func (bc *testBlockChain) InsertChain(chain types.Blocks, verifyHeaders bool) (int, error) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for i, block := range chain {
		if block.NumberU64() <= bc.curBN {
			continue
		}
		if block.NumberU64() != bc.curBN+1 {
			return i, errors.New("not expected block number")
		}
	}
	return len(chain), nil
}

func (bc *testBlockChain) ShardID() uint32 {
	return 0
}
