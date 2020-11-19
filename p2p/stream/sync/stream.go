package sync

import (
	"fmt"
	"sync"

	"github.com/harmony-one/harmony/p2p/stream/utils/requestmanager"

	"github.com/harmony-one/harmony/p2p/stream/utils/ratelimiter"
	"github.com/harmony-one/harmony/p2p/stream/utils/streammanager"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// SyncStream is the structure for a stream of a peer running sync protocol
type SyncStream struct {
	// Basic stream
	sttypes.BaseStream

	chain engine.ChainReader          // provide SYNC data
	rl    ratelimiter.RateLimiter     // limit the incoming request rate
	sm    streammanager.StreamManager // stream management
	rm    requestmanager.Deliverer    // deliver the response from stream

	// pipeline channels
	msgC chan *message.Message

	// close related fields. Multiple call of close is expected.
	closeC    chan struct{}
	closeOnce sync.Once
	closeErr  error

	logger zerolog.Logger
}

func (st *SyncStream) run() {
	var (
		inMsgCh = make(chan *message.Message)
	)
	defer st.Close()

	// convert read message to a channel
	go func() {
		for {
			msg, err := st.ReadMsg()
			if err != nil {
				// err from read message can happen two cases:
				// 1. The stream is terminated by the other side and got EOF.
				// 2. The stream is terminated by this side because an error
				//    happens when handling message
				st.Close()
			}
			inMsgCh <- msg.(*message.Message)
		}
	}()

	for {
		select {
		case <-st.closeC:
			return
		case msg := <-inMsgCh:
			err := st.handleMsg(msg)
			if err != nil {
				st.logger.Err(err).Msg("handle request")
				return
			}
		}
	}
}

// Close stops the stream handling and closes the underlying stream
func (st *SyncStream) Close() error {
	st.closeOnce.Do(func() {
		st.closeC <- struct{}{}
		close(st.closeC)
		st.closeErr = st.BaseStream.Close()
	})
	return st.closeErr
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
