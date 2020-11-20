package sync

import (
	"fmt"
	"sync/atomic"

	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// syncStream is the structure for a stream of a peer running sync protocol.
type syncStream struct {
	// Basic stream
	sttypes.BaseStream

	protocol *Protocol

	// pipeline channels
	msgC chan *message.Message

	// close related fields. Concurrent call of close is possible.
	closeC    chan struct{}
	closeStat uint32

	logger zerolog.Logger
}

// wrapStream wraps the raw libp2p stream to syncStream
func (p *Protocol) wrapStream(raw libp2p_network.Stream) *syncStream {
	bs := sttypes.NewBaseStream(raw)
	return &syncStream{
		BaseStream: *bs,
		protocol:   p,
		msgC:       make(chan *message.Message),
		closeC:     make(chan struct{}),
		closeStat:  0,
		logger: p.logger.With().
			Str("ID", string(bs.ID())).
			Str("Remote Protocol", string(bs.ProtoID())).
			Logger(),
	}
}

func (st *syncStream) run() {
	go st.readMsgLoop()
	st.handleMsgLoop()
}

func (st *syncStream) readMsgLoop() {
	for {
		msg, err := st.ReadMsg()
		if err != nil {
			if err := st.Close(); err != nil {
				st.logger.Err(err).Msg("failed to close sync stream")
			}
		}
		select {
		case st.msgC <- msg.(*message.Message):
		case <-st.closeC:
			return
		}
	}
}

func (st *syncStream) handleMsgLoop() {
	for {
		select {
		case msg := <-st.msgC:
			err := st.handleMsg(msg)
			if err != nil {
				st.logger.Err(err).Msg("handle request error")
				if err := st.Close(); err != nil {
					st.logger.Err(err).Msg("failed to close sync stream")
				}
				return
			}

		case <-st.closeC:
			return
		}
	}
}

// Close stops the stream handling and closes the underlying stream
func (st *syncStream) Close() error {
	// Hack here: Close is called in two cases:
	// 1. error happened when running the stream (readMsgLoop, handleMsgLoop)
	// 2. error happened when validating the result delivered by the stream, thus
	//    close the stream at stream manager.
	// Thus we only need to close for only once to prevent recursive call of the
	// Close method (syncStream -> StreamManager -> syncStream -> ...)
	notClosed := atomic.CompareAndSwapUint32(&st.closeStat, 0, 1)
	if !notClosed {
		// Already closed by another goroutine. Directly return
		return nil
	}
	err := st.BaseStream.Close()
	st.protocol.sm.RemoveStream(st.ID())
	close(st.closeC)
	return err
}

func (st *syncStream) handleMsg(msg *message.Message) error {
	if msg == nil {
		return nil
	}
	if req := msg.GetReq(); req != nil {
		st.protocol.rl.LimitRequest(st.ID(), req)
		return st.handleReq(req)
	}
	if resp := msg.GetResp(); resp != nil {
		st.handleResp(resp)
		return nil
	}
	return nil
}

func (st *syncStream) handleReq(req *message.Request) error {
	if bnReq := req.GetGetBlocksByNumRequest(); bnReq != nil {
		bns := req.GetGetBlocksByNumRequest().Nums
		resp, err := st.getRespFromBlockNumber(req.ReqId, bns)
		if err != nil {
			return errors.Wrap(err, "[GetBlocksByNumber]")
		}
		return st.WriteMsg(resp)
	}
	// unsupported request type
	resp := message.MakeErrorResponseMessage(errUnknownReqType)
	return st.WriteMsg(resp)
}

func (st *syncStream) handleResp(resp *message.Response) {
	st.protocol.rm.DeliverResponse(st.ID(), resp)
}

const (
	// GetBlocksByNumAmountCap is the cap of request of a single GetBlocksByNum request
	GetBlocksByNumAmountCap = 10
)

func (st *syncStream) getRespFromBlockNumber(rid uint64, bns []int64) (*message.Message, error) {
	if len(bns) > GetBlocksByNumAmountCap {
		return nil, fmt.Errorf("GetBlocksByNum amount exceed cap: %v/%v", len(bns), GetBlocksByNumAmountCap)
	}
	blocks := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		var block *types.Block
		header := st.protocol.chain.GetHeaderByNumber(uint64(bn))
		if header != nil {
			block = st.protocol.chain.GetBlock(header.Hash(), header.Number().Uint64())
		}
		blocks = append(blocks, block)
	}
	return message.MakeGetBlocksByNumResponse(rid, blocks)
}
