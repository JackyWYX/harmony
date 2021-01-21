package downloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/common/streammanager"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type worker struct {
	tm       *bootTaskManager
	protocol syncProtocol

	ctx context.Context
}

func (w *worker) start() {
	go w.workLoop()
}

func (w *worker) workLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}
		task := w.tm.getNextTask()
		if task == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		blocks, stid, err := w.doTask(task)
		if err != nil {
			w.tm.handleTaskErrorResult(task, stid, err)
		} else {
			w.tm.handleTaskResult(task, blocks, stid)
		}
	}
}

func (w *worker) doTask(task *getBlocksTask) ([]*types.Block, sttypes.StreamID, error) {
	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()
	blocks, stid, err := w.protocol.GetBlocksByNumber(ctx, task.bns)
	if err != nil {
		return nil, stid, err
	}
	if err := task.validateResult(blocks); err != nil {
		return nil, stid, errors.Wrap(err, "validate failed")
	}
	return blocks, stid, err
}

type (
	bootTaskManager struct {
		chain    blockChain
		sm       streammanager.StreamOperator
		protocol syncProtocol

		workers  []*worker
		pendings map[uint64]struct{} // block numbers that have been assigned to workers but not inserted
		retries  *prioritizedNumbers // requests where error happens
		rq       *resultQueue        // result queue wait to be inserted into blockchain
		resultC  chan struct{}

		ctx    context.Context
		cancel func()
		lock   sync.Mutex
		logger zerolog.Logger
	}
)

func (tm *bootTaskManager) start() {

	tm.workers = make([]*worker, 0, concurrency)
	for i := 0; i != concurrency; i++ {
		tm.workers = append(tm.workers, &worker{
			tm:       tm,
			protocol: tm.protocol,
			ctx:      tm.ctx,
		})
	}

	go tm.insertChainLoop()
	for _, worker := range tm.workers {
		go worker.start()
	}
}

// getNextTask get the next task assigned to workers.
// The strategy is as follows:
// 1. Get the block number from previous failed requests.
// 2. If there are enough space for chain insert, get the block number from unprocessed
// 3. Add the task to pendings and return the task.
func (tm *bootTaskManager) getNextTask() *getBlocksTask {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	cap := blocksPerRequest

	bns := tm.getTasksFromRetries(cap)
	cap -= len(bns)
	tm.addTasksToPending(bns)

	if tm.availableForMoreTasks() {
		addBNs := tm.getTasksFromUnprocessed(cap)
		tm.addTasksToPending(addBNs)
		bns = append(bns, addBNs...)
	}

	if len(bns) == 0 {
		return nil
	}

	return &getBlocksTask{
		bns: bns,
	}
}

// handleTaskResult handles the queried result from sync stream protocol
func (tm *bootTaskManager) handleTaskResult(task *getBlocksTask, blocks []*types.Block, stid sttypes.StreamID) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	for i, block := range blocks {
		if block == nil {
			// nil block hit. Remove from pendings
			delete(tm.pendings, task.bns[i])
		}
	}
	tm.rq.addBlockResults(blocks, stid)
	select {
	case tm.resultC <- struct{}{}:
	default:
	}
}

// handleTaskErrorResult handles the task where an error happens.
// Remove the stream and re-add the task to retries and removed from pendings
func (tm *bootTaskManager) handleTaskErrorResult(task *getBlocksTask, stid sttypes.StreamID, err error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.logger.Warn().Err(err).Str("streamID", string(stid)).
		Msg("error when doing getBlocksTask, removing stream")

	tm.handleBadStream(stid)
	for _, bn := range task.bns {
		tm.retries.push(bn)
		delete(tm.pendings, bn)
	}
}

func (tm *bootTaskManager) getTasksFromRetries(cap int) []uint64 {
	var (
		requestBNs []uint64
		curHeight  = tm.chain.CurrentBlock().NumberU64()
	)
	for cnt := 0; cnt < cap; cnt++ {
		bn := tm.retries.pop()
		if bn == 0 {
			break // no more retries
		}
		if bn <= curHeight {
			continue
		}
		requestBNs = append(requestBNs, bn)
	}
	return requestBNs
}

func (tm *bootTaskManager) getTasksFromUnprocessed(cap int) []uint64 {
	var (
		requestBNs []uint64
		curHeight  = tm.chain.CurrentBlock().NumberU64()
	)
	bn := curHeight + 1
	// TODO: this algorithm can be potentially optimized.
	for cnt := 0; cnt < cap; {
		for {
			if _, ok := tm.pendings[bn]; !ok {
				requestBNs = append(requestBNs, bn)
				cnt++
				bn++
				break
			}
			bn++
		}
	}
	return requestBNs
}

func (tm *bootTaskManager) availableForMoreTasks() bool {
	return tm.rq.results.Len() < softQueueCap
}

func (tm *bootTaskManager) addTasksToPending(bns []uint64) {
	for _, bn := range bns {
		tm.pendings[bn] = struct{}{}
	}
}

// insertChainLoop constantly pull blocks from resultQueue and insert it into chain.
func (tm *bootTaskManager) insertChainLoop() {
	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-t.C:
			// Redundancy, periodically check whether there is blocks to be processed
			select {
			case tm.resultC <- struct{}{}:
			default:
			}
		case <-tm.resultC:
			blockResults := tm.pullContinuousBlocks(blocksPerBatch)
			if len(blockResults) > 0 {
				tm.processBlocks(blockResults)
			}
		}
	}
}

// pullContinuousBlocks pull continuous blocks from the result queue
func (tm *bootTaskManager) pullContinuousBlocks(cap int) []*blockResult {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	expHeight := tm.chain.CurrentBlock().NumberU64() + 1
	results, stales := tm.rq.popBlockResults(expHeight, cap)
	// For stale blocks, we remove them from pendings
	for _, bn := range stales {
		delete(tm.pendings, bn)
	}
	return results
}

func (tm *bootTaskManager) processBlocks(results []*blockResult) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	for i, result := range results {
		if _, err := tm.chain.InsertChain([]*types.Block{result.block}, true); err != nil {
			if i != len(results)-1 {
				remains := results[i+1:]
				for _, remain := range remains {
					tm.rq.addBlockResults([]*types.Block{remain.block}, remain.stid)
				}
			}
			tm.handleBadStream(result.stid)
			break
		}
	}
	return
}

// handleBadStream handles a stream delivering a bad response.
// 1. remove the stream from stream manager
// 2. remove all results from result queue and add them to retries
func (tm *bootTaskManager) handleBadStream(stid sttypes.StreamID) {
	if err := tm.sm.RemoveStream(stid); err != nil {
		tm.logger.Warn().Err(err).Str("stream", string(stid)).
			Msg("failed to remove stream")
	}
	removed := tm.rq.removeResultsByStreamID(stid)
	for _, retry := range removed {
		tm.pendings[retry] = struct{}{}
	}
}

type getBlocksTask struct {
	bns []uint64
}

func (task *getBlocksTask) validateResult(blocks []*types.Block) error {
	if len(blocks) != len(task.bns) {
		return fmt.Errorf("unexpected number of blocks delivered: %v / %v", len(blocks), len(task.bns))
	}
	for i, block := range blocks {
		if block != nil && block.NumberU64() != task.bns[i] {
			return fmt.Errorf("block with unexpected number delivered: %v / %v", block.NumberU64(), task.bns[i])
		}
	}
	return nil
}
