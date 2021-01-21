package downloader

import (
	"testing"
)

func TestBootTaskManager_getNextTask(t *testing.T) {
	tests := []struct {
		tm     *bootTaskManager
		expBNs []uint64
	}{
		{
			tm: makeBootTaskManager(
				10,
				[]uint64{9, 11, 12, 13},
				[]uint64{},
				0,
			),
			expBNs: []uint64{14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		},
		{
			tm: makeBootTaskManager(
				10,
				[]uint64{9, 13, 14, 15, 16},
				[]uint64{10, 11, 12},
				0,
			),
			expBNs: []uint64{11, 12, 17, 18, 19, 20, 21, 22, 23, 24},
		},
		{
			tm: makeBootTaskManager(
				10,
				[]uint64{9, 13, 14, 15, 16},
				[]uint64{10, 11, 12},
				120,
			),
			expBNs: []uint64{11, 12},
		},
		{
			tm: makeBootTaskManager(
				10,
				[]uint64{9, 13, 14, 15, 16},
				[]uint64{},
				120,
			),
			expBNs: nil,
		},
	}

	for i, test := range tests {
		task := test.tm.getNextTask()
		if len(test.expBNs) == 0 && task != nil {
			t.Errorf("Test %v: expect nil task", i)
		}
		if len(test.expBNs) == 0 {
			continue
		}

		if len(test.expBNs) != len(task.bns) {
			t.Errorf("Test %v: unexpected size [%v] / [%v]", i, task.bns, test.expBNs)
		}
		for i := range test.expBNs {
			if test.expBNs[i] != task.bns[i] {
				t.Errorf("Test %v: [%v] / [%v]", i, task.bns, test.expBNs)
			}
		}
	}
}

func makeBootTaskManager(curBN uint64, pendings, retries []uint64, sizeRQ int) *bootTaskManager {
	chain := newTestBlockChain(curBN)
	pendingsM := make(map[uint64]struct{})
	for _, pending := range pendings {
		pendingsM[pending] = struct{}{}
	}
	retriesPN := newPrioritizedNumbers()
	for _, retry := range retries {
		retriesPN.push(retry)
	}
	rq := newResultQueue()
	for i := uint64(0); i != uint64(sizeRQ); i++ {
		rq.addBlockResults(makeTestBlocks([]uint64{100}), "")
	}
	return &bootTaskManager{
		chain:    chain,
		pendings: pendingsM,
		retries:  retriesPN,
		rq:       rq,
	}
}
