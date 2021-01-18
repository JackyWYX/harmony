package downloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/harmony-one/harmony/consensus/engine"

	"github.com/rs/zerolog"

	"github.com/harmony-one/harmony/core/types"
	"github.com/pkg/errors"
)

type worker struct {
	tm       taskManager
	protocol syncProtocol
	closeC   chan struct{}

	ctx    context.Context
	cancel func()
}

func (w *worker) run() {
	go w.workLoop()
}

func (w *worker) workLoop() {
	for {
		task := w.tm.getNextTask()
		if task == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		res, err := w.doTask(task)
		if err != nil {
			w.tm.handleTaskErrorResult(task, res, err)
		} else {
			w.tm.handleTaskResult(task, res)
		}
	}
}

func (w *worker) doTask(task *getBlocksTask) (*getBlocksResult, error) {
	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()
	blocks, stid, err := w.protocol.GetBlocksByNumber(ctx, task.bns)
	if err != nil {
		return nil, err
	}
	if err := task.validateResult(blocks); err != nil {
		return nil, errors.Wrap(err, "validate failed")
	}
	return &getBlocksResult{
		bns:    task.bns,
		blocks: blocks,
		stid:   stid,
	}, err
}

type (
	taskManager struct {
		chain engine.ChainReader

		pendings map[uint64]struct{}
		retries  *prioritizedNumbers
		rq       *resultQueue

		lock   sync.Mutex
		logger zerolog.Logger
	}
)

func (tm *taskManager) getNextTask() *getBlocksTask {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	curHeight := tm.chain.CurrentBlock().NumberU64()
	requestBNs := make([]uint64, 0, defaultBlocksPerRequest)
	cnt := 0
	// First get number from previously failed blocks
	for cnt < defaultBlocksPerRequest {
		bn := tm.retries.pop()
		if bn == 0 {
			break
		}
		if bn <= curHeight {
			continue
		}
		requestBNs = append(requestBNs, bn)
		cnt++
	}
	// If there is already plenty of blocks to be processed in result queue,
	// do not add more tasks
	if tm.rq.results.Len() >= softQueueCap {
		for _, bn := range requestBNs {
			tm.pendings[bn] = struct{}{}
		}
		return &getBlocksTask{
			bns: requestBNs,
		}
	}
	// Then get number from future blocks which is not in pendings
	bn := curHeight + 1
	for cnt < defaultBlocksPerRequest {
		for ; ; bn++ {
			if _, ok := tm.pendings[bn]; !ok {
				requestBNs = append(requestBNs, bn)
				cnt++
				break
			}
		}
	}
	// add the bns to pendings
	for _, bn := range requestBNs {
		tm.pendings[bn] = struct{}{}
	}
	return &getBlocksTask{
		bns: requestBNs,
	}
}

func (tm *taskManager) handleTaskResult(task *getBlocksTask, res *getBlocksResult) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	return
}

func (tm *taskManager) handleTaskErrorResult(task *getBlocksTask, res *getBlocksResult, err error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	return
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
