package explorer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/harmony-one/harmony/core"
	"github.com/harmony-one/harmony/core/types"
)

type InsertHelper interface {
	Dump(b *types.Block)
}

type explorerHelper struct {
	bc *core.BlockChain

	lastClean     map[common.Address][]byte // last DB commit
	lastCommitNum uint64

	blockC chan *types.Block
	closeC chan struct{}
}

func (helper *explorerHelper) Dump(b *types.Block) {
	helper.blockC <- b
}

func newExplorerHelper() *explorerHelper {
	return &explorerHelper{}
}

func (helper *explorerHelper) Start() {
	go helper.loop()
}

func (helper *explorerHelper) Stop() {
	close(helper.closeC)
}

func (helper *explorerHelper) loop() {
	computeC := make(chan *types.Block, 1)
	computeTrigger := func(b *types.Block) {
		select {
		case computeC <- b:
		default:
		}
	}

	resultC := make(chan []*Address, 1)
	resultTrigger := func(addrs []*Address) {
		select {
		case resultC <- addrs:
		default:
		}
	}

	for {
		select {
		case b := <-helper.blockC:
			number := b.NumberU64()
			if number < helper.lastCommitNum+10 {
				continue
			}
			helper.lastCommitNum = number
			computeTrigger(b)

		case b := <-computeC:
			go func(b *types.Block) {
				result := helper.computeResult()
				resultTrigger(result)
			}(b)

		case res := <-resultC:
			err := helper.commitResult(res)
			if err != nil {
				// report error
			}

		case <-helper.closeC:
			// log
			break
		}
	}
	return
}

func (helper *explorerHelper) computeResult() map[common.Address][]byte {
	// fill
}

func (helper *explorerHelper) commitResult(res map[common.Address][]byte) error {
	// DB write
}
