package downloader

import (
	"container/heap"
	"sync"

	"github.com/harmony-one/harmony/core/types"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
)

var (
	errResultQueueFull = errors.New("resultQueue is already full.")
)

type getBlocksResult struct {
	bns    []uint64
	blocks []*types.Block
	stid   sttypes.StreamID
}

type resultQueue struct {
	results *priorityQueue
	lock    sync.Mutex
}

func newResultQueue() *resultQueue {
	pq := make(priorityQueue, 0, queueMaxSize)
	heap.Init(&pq)
	return &resultQueue{
		results: &pq,
	}
}

func (rq *resultQueue) addBlockResults(blocks []*types.Block, stid sttypes.StreamID) error {
	rq.lock.Lock()
	defer rq.lock.Unlock()

	if len(blocks)+rq.results.Len() > queueMaxSize {
		return errResultQueueFull
	}
	for _, block := range blocks {
		if block == nil {
			return nil
		}
		heap.Push(rq.results, &blockResult{
			block: block,
			stid:  stid,
		})
	}
	return nil
}

func (rq *resultQueue) popBlockResults(expStartBN uint64, cap int) []*blockResult {
	rq.lock.Lock()
	defer rq.lock.Unlock()

	res := make([]*blockResult, 0, cap)
	for cnt := 0; rq.results.Len() > 0 && cnt < cap; cnt++ {
		br := heap.Pop(rq.results).(*blockResult)

		if br.block.NumberU64() < expStartBN {
			// duplicate block number or low starting number, skipping
			continue
		}
		if br.block.NumberU64() != expStartBN {
			return res
		}
		res = append(res, br)
		expStartBN++
	}
	return res
}

// removeResultsByStreamID remove the block results of the given stream, return the block
// number removed from the queue
func (rq *resultQueue) removeResultsByStreamID(stid sttypes.StreamID) []uint64 {
	rq.lock.Lock()
	defer rq.lock.Unlock()

	var removed []uint64

Loop:
	for {
		for i, res := range *rq.results {
			blockRes := res.(*blockResult)
			if blockRes.stid == stid {
				heap.Remove(rq.results, i)
				removed = append(removed, blockRes.block.NumberU64())
				goto Loop
			}
		}
		break
	}
	return removed
}

func (rq *resultQueue) removeByIndex(index int) {
	heap.Remove(rq.results, index)
}

// bnPrioritizedItem is the item which uses block number to determine its priority
type bnPrioritizedItem interface {
	getBlockNumber() uint64
}

type blockResult struct {
	block *types.Block
	stid  sttypes.StreamID
}

func (br *blockResult) getBlockNumber() uint64 {
	return br.block.NumberU64()
}

type (
	prioritizedNumber uint64

	prioritizedNumbers struct {
		q *priorityQueue
	}
)

func (b prioritizedNumber) getBlockNumber() uint64 {
	return uint64(b)
}

func newPrioritizedNumbers() *prioritizedNumbers {
	pqs := make(priorityQueue, 0)
	heap.Init(&pqs)
	return &prioritizedNumbers{
		q: &pqs,
	}
}

func (pbs *prioritizedNumbers) push(bn uint64) {
	heap.Push(pbs.q, prioritizedNumber(bn))
}

func (pbs *prioritizedNumbers) pop() uint64 {
	if pbs.q.Len() == 0 {
		return 0
	}
	item := heap.Pop(pbs.q)
	return uint64(item.(prioritizedNumber))
}

// priorityQueue is a priorityQueue with lowest block number with highest priority
type priorityQueue []bnPrioritizedItem

// resultQueue implements heap interface
func (q priorityQueue) Len() int {
	return len(q)
}

// resultQueue implements heap interface
func (q priorityQueue) Less(i, j int) bool {
	bn1 := q[i].getBlockNumber()
	bn2 := q[j].getBlockNumber()
	return bn1 < bn2 // small block number has higher priority
}

// resultQueue implements heap interface
func (q priorityQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

// resultQueue implements heap interface
func (q *priorityQueue) Push(x interface{}) {
	item, ok := x.(bnPrioritizedItem)
	if !ok {
		panic("wrong type of getBlockNumber interface")
	}
	*q = append(*q, item)
}

// resultQueue implements heap interface
func (q *priorityQueue) Pop() interface{} {
	prev := *q
	n := len(prev)
	res := prev[n-1]
	*q = prev[0 : n-1]
	return res
}
