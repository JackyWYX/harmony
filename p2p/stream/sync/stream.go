package sync

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
)

// SyncStream is the structure for a stream of a peer running a single protocol
type SyncStream struct {
	sttypes.BaseStream // extends the basic stream

	chain engine.ChainReader // provide SYNC data

	rl sttypes.RateLimiter // limit the incoming request rate

	deliverer deliverer

	stopCh chan struct{}
}

type deliverer interface {
	deliverBlocks(st *SyncStream, blocks []*types.Block)
}

func (st *SyncStream) run() {
	var (
		inMsgCh = make(chan *message.Message)
	)
	defer st.stop()

	// convert read message to a channel
	go func() {
		for {
			msg, err := st.ReadMsg()
			if err != nil {
				st.stop()
			}
			inMsgCh <- msg.(*message.Message)
		}
	}()

	for {
		select {
		case <-st.stopCh:
			return
		case msg := <-inMsgCh:
			err := st.handleMsg(msg)
			if err != nil {
				return
			}
		}
	}
}

func (st *SyncStream) stop() {
	close(st.stopCh)
	st.Close()
}

func (st *SyncStream) handleMsg(msg *message.Message) error {
	if msg == nil {
		return nil
	}
	if req := msg.GetReq(); req != nil {
		st.rl.LimitRequest(st, req)
		return st.handleReq(req)
	}
	if resp := msg.GetResp(); resp != nil {
		return st.handleResp(resp)
	}
	return nil
}

func (st *SyncStream) handleReq(req *message.Request) error {
	if bnReq := req.GetGetBlocksByNumRequest(); bnReq != nil {
		resp, err := st.getRespFromBlockNumber(req.ReqId, req.GetGetBlocksByNumRequest().Nums)
		if err != nil {
			return errors.Wrap(err, "[GetBlocksByNumber]")
		}
		return st.WriteMsg(resp)
	}
	return nil
}

func (st *SyncStream) handleResp(resp *message.Response) error {
	if errResp := resp.GetErrorResponse(); errResp != nil {
		return errors.New(errResp.Error)
	}
	if gbResp := resp.GetGetBlocksByNumResponse(); gbResp != nil {
		blocks := make([]*types.Block, 0, len(gbResp.Blocks))
		for _, bb := range gbResp.Blocks {
			var block *types.Block
			if err := rlp.DecodeBytes(bb, &block); err != nil {
				return errors.Wrap(err, "[GetBlocksByNumResponse]")
			}
			blocks = append(blocks, block)
		}
		st.deliverer.deliverBlocks(st, blocks)
	}
	return nil
}

const (
	// GetBlocksByNumAmountCap is the cap of request of a single GetBlocksByNum request
	GetBlocksByNumAmountCap = 10
)

func (st *SyncStream) getRespFromBlockNumber(rid uint64, bns []int64) (*message.Message, error) {
	if len(bns) > GetBlocksByNumAmountCap {
		return nil, fmt.Errorf("GetBlocksByNum amount exceed cap: %v/%v", len(bns), GetBlocksByNumAmountCap)
	}
	blocks := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		var block *types.Block
		header := st.chain.GetHeaderByNumber(uint64(bn))
		if header != nil {
			block = st.chain.GetBlock(header.Hash(), header.Number().Uint64())
		}
		blocks = append(blocks, block)
	}
	return message.MakeGetBlocksByNumResponse(rid, blocks)
}
