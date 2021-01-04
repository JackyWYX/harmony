package sync

import (
	"fmt"
	"sync/atomic"
	"time"

	protobuf "github.com/golang/protobuf/proto"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/sync/syncpb"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

// syncStream is the structure for a stream running sync protocol.
type syncStream struct {
	// Basic stream
	*sttypes.BaseStream

	protocol *Protocol

	// pipeline channels
	reqC  chan *syncpb.Request
	respC chan *syncpb.Response

	// close related fields. Concurrent call of close is possible.
	closeC    chan struct{}
	closeStat uint32

	logger zerolog.Logger
}

// wrapStream wraps the raw libp2p stream to syncStream
func (p *Protocol) wrapStream(raw libp2p_network.Stream) *syncStream {
	bs := sttypes.NewBaseStream(raw)
	logger := p.logger.With().
		Str("ID", string(bs.ID())).
		Str("Remote Protocol", string(bs.ProtoID())).
		Logger()

	return &syncStream{
		BaseStream: bs,
		protocol:   p,
		reqC:       make(chan *syncpb.Request, 100),
		respC:      make(chan *syncpb.Response, 100),
		closeC:     make(chan struct{}),
		closeStat:  0,
		logger:     logger,
	}
}

func (st *syncStream) run() {
	go st.readMsgLoop()
	go st.handleReqLoop()
	go st.handleRespLoop()
}

// readMsgLoop is the loop
func (st *syncStream) readMsgLoop() {
	for {
		msg, err := st.ReadMsg()
		if err != nil {
			if err := st.Close(); err != nil {
				st.logger.Err(err).Msg("failed to close sync stream")
			}
		}
		st.deliverMsg(msg)
	}
}

// deliverMsg process the delivered message and forward to the corresponding channel
func (st *syncStream) deliverMsg(msg protobuf.Message) {
	syncMsg := msg.(*syncpb.Message)
	if syncMsg == nil {
		st.logger.Info().Str("message", msg.String()).Msg("received unexpected sync message")
		return
	}
	if req := syncMsg.GetReq(); req != nil {
		go func() {
			select {
			case st.reqC <- req:
			case <-time.After(1 * time.Minute):
				st.logger.Warn().Str("request", req.String()).
					Msg("request handler severely jammed, message dropped")
			}
		}()
	}
	if resp := syncMsg.GetResp(); resp != nil {
		go func() {
			select {
			case st.respC <- resp:
			case <-time.After(1 * time.Minute):
				st.logger.Warn().Str("response", resp.String()).
					Msg("response handler severely jammed, message dropped")
			}
		}()
	}
	return
}

func (st *syncStream) handleReqLoop() {
	for {
		select {
		case req := <-st.reqC:
			st.protocol.rl.LimitRequest(st.ID(), req)
			err := st.handleReq(req)

			if err != nil {
				st.logger.Info().Err(err).Str("request", req.String()).
					Msg("handle request error. Closing stream")
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

func (st *syncStream) handleRespLoop() {
	for {
		select {
		case resp := <-st.respC:
			st.handleResp(resp)

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
	if err := st.protocol.sm.RemoveStream(st.ID()); err != nil {
		st.logger.Err(err).Str("stream ID", string(st.ID())).
			Msg("failed to remove sync stream on close")
	}
	close(st.closeC)
	return err
}

func (st *syncStream) handleReq(req *syncpb.Request) error {

	if bnReq := req.GetGetBlocksByNumRequest(); bnReq != nil {
		resp, err := st.getRespFromBlockNumber(req.ReqId, bnReq.Nums)
		if resp == nil && err != nil {
			resp = syncpb.MakeErrorResponseMessage(req.ReqId, err)
		}
		if writeErr := st.WriteMsg(resp); writeErr != nil {
			if err == nil {
				err = writeErr
			} else {
				err = fmt.Errorf("%v; [writeMsg] %v", err.Error(), writeErr)
			}
		}
		return errors.Wrap(err, "[GetBlocksByNumber]")
	}
	// unsupported request type
	resp := syncpb.MakeErrorResponseMessage(req.ReqId, errUnknownReqType)
	return st.WriteMsg(resp)
}

func (st *syncStream) handleResp(resp *syncpb.Response) {
	st.protocol.rm.DeliverResponse(st.ID(), &syncResponse{resp})
}

const (
	// GetBlocksByNumAmountCap is the cap of request of a single GetBlocksByNum request
	GetBlocksByNumAmountCap = 10
)

func (st *syncStream) getRespFromBlockNumber(rid uint64, bns []uint64) (*syncpb.Message, error) {
	if len(bns) > GetBlocksByNumAmountCap {
		err := fmt.Errorf("GetBlocksByNum amount exceed cap: %v/%v", len(bns), GetBlocksByNumAmountCap)
		return nil, err
	}
	blocks := make([]*types.Block, 0, len(bns))
	for _, bn := range bns {
		var block *types.Block
		header := st.protocol.chain.GetHeaderByNumber(bn)
		if header != nil {
			block = st.protocol.chain.GetBlock(header.Hash(), header.Number().Uint64())
		}
		blocks = append(blocks, block)
	}
	return syncpb.MakeGetBlocksByNumResponse(rid, blocks)
}
