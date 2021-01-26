package downloader

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"

	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

func TestLrSyncIter_EstimateCurrentNumber(t *testing.T) {
	lsi := &lrSyncIter{
		protocol: newTestSyncProtocol(100),
		ctx:      context.Background(),
		config: Config{
			Concurrency: 16,
			MinStreams:  10,
		},
	}
	bn, err := lsi.estimateCurrentNumber()
	if err != nil {
		t.Error(err)
	}
	if bn != 100 {
		t.Errorf("unexpected block number: %v / %v", bn, 100)
	}
}

func TestGetBlocksManager_GetNextBatch(t *testing.T) {
	tests := []struct {
		gbm    *getBlocksManager
		expBNs []uint64
	}{
		{
			gbm: makeGetBlocksManager(
				10, 100, []uint64{9, 11, 12, 13},
				[]uint64{14, 15, 16}, []uint64{}, 0,
			),
			expBNs: []uint64{17, 18, 19, 20, 21, 22, 23, 24, 25, 26},
		},
		{
			gbm: makeGetBlocksManager(
				10, 100, []uint64{9, 13, 14, 15, 16},
				[]uint64{}, []uint64{10, 11, 12}, 0,
			),
			expBNs: []uint64{11, 12, 17, 18, 19, 20, 21, 22, 23, 24},
		},
		{
			gbm: makeGetBlocksManager(
				10, 100, []uint64{9, 13, 14, 15, 16},
				[]uint64{}, []uint64{10, 11, 12}, 120,
			),
			expBNs: []uint64{11, 12},
		},
		{
			gbm: makeGetBlocksManager(
				10, 100, []uint64{9, 13, 14, 15, 16},
				[]uint64{}, []uint64{}, 120,
			),
			expBNs: []uint64{},
		},
		{
			gbm: makeGetBlocksManager(
				10, 20, []uint64{9, 13, 14, 15, 16},
				[]uint64{}, []uint64{}, 0,
			),
			expBNs: []uint64{11, 12, 17, 18, 19, 20},
		},
		{
			gbm: makeGetBlocksManager(
				10, 100, []uint64{9, 13, 14, 15, 16},
				[]uint64{}, []uint64{}, 0,
			),
			expBNs: []uint64{11, 12, 17, 18, 19, 20, 21, 22, 23, 24},
		},
	}

	for i, test := range tests {
		if i < 4 {
			continue
		}
		batch := test.gbm.GetNextBatch()
		if len(test.expBNs) != len(batch) {
			t.Errorf("Test %v: unexpected size [%v] / [%v]", i, batch, test.expBNs)
		}
		fmt.Println(batch)
		for i := range test.expBNs {
			if test.expBNs[i] != batch[i] {
				t.Errorf("Test %v: [%v] / [%v]", i, batch, test.expBNs)
			}
		}
	}
}

func TestLrSyncIter_FetchAndInsertBlocks(t *testing.T) {
	targetBN := uint64(1000)
	chain := newTestBlockChain(0)
	protocol := newTestSyncProtocol(targetBN)
	ctx, cancel := context.WithCancel(context.Background())

	lsi := &lrSyncIter{
		chain:    chain,
		protocol: protocol,
		gbm:      nil,
		config: Config{
			Concurrency: 100,
		},
		ctx:    ctx,
		cancel: cancel,
		logger: zerolog.New(os.Stdout),
	}
	lsi.fetchAndInsertBlocks(targetBN)

	time.Sleep(100 * time.Millisecond)

	if bn := chain.currentBlockNumber(); bn != targetBN {
		t.Errorf("did not reached targetBN: %v / %v", bn, targetBN)
	}
	if len(lsi.gbm.processing) != 0 {
		t.Errorf("not empty processing")
	}
	if len(lsi.gbm.requesting) != 0 {
		t.Errorf("not empty requesting")
	}
	if lsi.gbm.retries.length() != 0 {
		t.Errorf("not empty retries")
	}
	if lsi.gbm.rq.length() != 0 {
		t.Errorf("not empty result queue")
	}
}

func TestComputeBNMaxVote(t *testing.T) {
	tests := []struct {
		votes map[sttypes.StreamID]uint64
		exp   uint64
	}{
		{
			votes: map[sttypes.StreamID]uint64{
				makeStreamID(0): 10,
				makeStreamID(1): 10,
				makeStreamID(2): 20,
			},
			exp: 10,
		},
		{
			votes: map[sttypes.StreamID]uint64{
				makeStreamID(0): 10,
				makeStreamID(1): 20,
			},
			exp: 20,
		},
		{
			votes: map[sttypes.StreamID]uint64{
				makeStreamID(0): 20,
				makeStreamID(1): 10,
				makeStreamID(2): 20,
			},
			exp: 20,
		},
	}

	for i, test := range tests {
		res := computeBNMaxVote(test.votes)
		if res != test.exp {
			t.Errorf("Test %v: unexpected bn %v / %v", i, res, test.exp)
		}
	}
}

func makeGetBlocksManager(curBN, targetBN uint64, requesting, processing, retries []uint64, sizeRQ int) *getBlocksManager {
	chain := newTestBlockChain(curBN)
	requestingM := make(map[uint64]struct{})
	for _, bn := range requesting {
		requestingM[bn] = struct{}{}
	}
	processingM := make(map[uint64]struct{})
	for _, bn := range processing {
		processingM[bn] = struct{}{}
	}
	retriesPN := newPrioritizedNumbers()
	for _, retry := range retries {
		retriesPN.push(retry)
	}
	rq := newResultQueue()
	for i := uint64(0); i != uint64(sizeRQ); i++ {
		rq.addBlockResults(makeTestBlocks([]uint64{uint64(i) + curBN}), "")
	}
	return &getBlocksManager{
		chain:      chain,
		targetBN:   targetBN,
		requesting: requestingM,
		processing: processingM,
		retries:    retriesPN,
		rq:         rq,
		resultC:    make(chan struct{}, 1),
	}
}
