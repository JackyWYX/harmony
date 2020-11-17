package requestmanager

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/event"
	"github.com/harmony-one/harmony/p2p/stream/message"
	"github.com/harmony-one/harmony/p2p/stream/streammanager"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
)

var testProtoID = sttypes.ProtoID("harmony/sync/unitest/0/1.0.0")

type testStreamManager struct {
	newStreamFeed event.Feed
	rmStreamFeed  event.Feed
}

func newTestStreamManager() *testStreamManager {
	return &testStreamManager{}
}

func (sm *testStreamManager) addNewStream(st sttypes.Stream) {
	sm.newStreamFeed.Send(streammanager.EvtStreamAdded{Stream: st})
}

func (sm *testStreamManager) rmStream(stid sttypes.StreamID) {
	sm.rmStreamFeed.Send(streammanager.EvtStreamRemoved{ID: stid})
}

func (sm *testStreamManager) SubscribeAddStreamEvent(ch chan<- streammanager.EvtStreamAdded) event.Subscription {
	return sm.newStreamFeed.Subscribe(ch)
}

func (sm *testStreamManager) SubscribeRemoveStreamEvent(ch chan<- streammanager.EvtStreamRemoved) event.Subscription {
	return sm.rmStreamFeed.Subscribe(ch)
}

type testStream struct {
	id      sttypes.StreamID
	rm      *requestManager
	deliver func(req *message.Request) // use goroutine inside this function
}

func (st *testStream) ID() sttypes.StreamID {
	return st.id
}

func (st *testStream) PeerID() p2ptypes.PeerID {
	return st.id.PeerID
}

func (st *testStream) ProtoID() sttypes.ProtoID {
	return st.id.ProtoID
}

func (st *testStream) SendRequest(req *message.Request) error {
	if st.rm != nil && st.deliver != nil {
		st.deliver(req)
	}
	return nil
}

func makeStreamID(index int) sttypes.StreamID {
	return sttypes.StreamID{
		PeerID:  makePeerID(index),
		ProtoID: testProtoID,
	}
}

func makePeerID(index int) p2ptypes.PeerID {
	return p2ptypes.PeerID(strconv.Itoa(index))
}

type testRequest struct {
	reqID uint64
	index int
}

func makeTestRequest(index int) *testRequest {
	return &testRequest{
		reqID: 0,
		index: index,
	}
}

func (req *testRequest) ReqID() uint64 {
	return req.reqID
}

func (req *testRequest) SetReqID(rid uint64) {
	req.reqID = rid
}

func (req *testRequest) String() string {
	return fmt.Sprintf("test request %v", req.index)
}

func (req *testRequest) GetRequestMessage() *message.Request {
	return &message.Request{
		ReqId: req.reqID,
	}
}

func (req *testRequest) getResponse() *message.Response {
	return &message.Response{
		ReqId: req.reqID,
	}
}
