package requestmanager

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	protobuf "github.com/golang/protobuf/proto"

	"github.com/harmony-one/harmony/p2p/stream/sync/syncpb"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
)

var (
	defTestSleep = 50 * time.Millisecond
)

// Request is delivered right away as expected
func TestRequestManager_Request_Normal(t *testing.T) {
	delayF := makeDefaultDelayFunc(150 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	res := <-ts.rm.doRequestAsync(ctx, makeTestRequest(0))

	if res.Err != nil {
		t.Errorf("unexpected error: %v", res.Err)
		return
	}
	if err := checkResponseMessage(res.Raw, testMsg); err != nil {
		t.Error(err)
	}
	if res.StID == "" {
		t.Errorf("unexpected stid")
	}
}

// The request is canceled by context
func TestRequestManager_Request_Cancel(t *testing.T) {
	delayF := makeDefaultDelayFunc(500 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))

	time.Sleep(defTestSleep)
	cancel()

	res := <-resC
	if res.Err != context.Canceled {
		t.Errorf("unexpected error: %v", res.Err)
	}
	if res.StID != "" {
		t.Errorf("unexpected stid")
	}
}

// request is timed out and retried
func TestRequestManager_Request_Retry(t *testing.T) {
	// Block for first try and
	delayF := makeOnceBlockDelayFunc(150 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)

	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))

	time.Sleep(defTestSleep)

	res := <-resC
	if res.Err != nil {
		t.Errorf("unexpected error: %v", res.Err)
	}
	if err := checkResponseMessage(res.Raw, testMsg); err != nil {
		t.Error(err)
	}
	if res.StID == "" {
		t.Errorf("unexpected stid")
	}
}

// error happens when adding request to waiting list
func TestRequestManager_NewStream(t *testing.T) {
	delayF := makeDefaultDelayFunc(500 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ts.sm.addNewStream(ts.makeTestStream(3))

	time.Sleep(defTestSleep)

	ts.rm.lock.Lock()
	if len(ts.rm.streams) != 4 || len(ts.rm.available) != 4 {
		t.Errorf("unexpected stream size")
	}
	ts.rm.lock.Unlock()
}

// For request assigned to the stream being removed, the request will be rescheduled.
func TestRequestManager_RemoveStream(t *testing.T) {
	delayF := makeOnceBlockDelayFunc(150 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))
	time.Sleep(defTestSleep)

	// remove the stream which is responsible for the request
	idToRemove := ts.pickOneOccupiedStream()
	ts.sm.rmStream(idToRemove)

	// the request is rescheduled thus there is supposed to be no errors
	res := <-resC
	if res.Err != nil {
		t.Errorf("unexpected error: %v", res.Err)
	}
	if err := checkResponseMessage(res.Raw, testMsg); err != nil {
		t.Error(err)
	}
	if res.StID == "" {
		t.Errorf("unexpected stid")
	}

	ts.rm.lock.Lock()
	if len(ts.rm.streams) != 2 || len(ts.rm.available) != 2 {
		t.Errorf("unexpected stream size")
	}
	ts.rm.lock.Unlock()
}

// stream delivers an unknown request ID
func TestRequestManager_UnknownDelivery(t *testing.T) {
	delayF := makeDefaultDelayFunc(150 * time.Millisecond)
	respF := func(req *syncpb.Request) *syncpb.Response {
		var rid uint64
		for rid == req.ReqId {
			rid++
		}
		return &syncpb.Response{
			ReqId:    rid,
			Response: nil,
		}
	}
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))
	time.Sleep(6 * time.Second)
	cancel()

	// Since the reqID is not delivered, the result is not delivered to the request
	// and be canceled
	res := <-resC
	if res.Err != context.Canceled {
		t.Errorf("unexpected error: %v", res.Err)
	}
}

// stream delivers a response for a canceled request
func TestRequestManager_StaleDelivery(t *testing.T) {
	delayF := makeDefaultDelayFunc(1 * time.Second)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()
	defer ts.Close()

	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))
	time.Sleep(2 * time.Second)

	// Since the reqID is not delivered, the result is not delivered to the request
	// and be canceled
	res := <-resC
	if res.Err != context.DeadlineExceeded {
		t.Errorf("unexpected error: %v", res.Err)
	}
}

// closing request manager will also close all
func TestRequestManager_Close(t *testing.T) {
	delayF := makeDefaultDelayFunc(1 * time.Second)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 3)
	ts.Start()

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	resC := ts.rm.doRequestAsync(ctx, makeTestRequest(0))
	time.Sleep(100 * time.Millisecond)
	ts.Close()

	// Since the reqID is not delivered, the result is not delivered to the request
	// and be canceled
	res := <-resC
	if assErr := assertError(res.Err, errors.New("request manager module closed")); assErr != nil {
		t.Errorf("unexpected error: %v", assErr)
	}
}

// test the race condition by spinning up a lot of goroutines
func TestRequestManager_Concurrency(t *testing.T) {
	var (
		testDuration = 10 * time.Second
		numThreads   = 25
	)
	delayF := makeDefaultDelayFunc(100 * time.Millisecond)
	respF := makeDefaultResponseFunc(testMsg)
	ts := newTestSuite(delayF, respF, 18)
	ts.Start()

	stopC := make(chan struct{})
	var (
		aErr    atomic.Value
		numReqs uint64
		wg      sync.WaitGroup
	)
	wg.Add(numThreads)
	for i := 0; i != numThreads; i++ {
		go func() {
			defer wg.Done()
			for {
				resC := ts.rm.doRequestAsync(context.Background(), makeTestRequest(1000))
				select {
				case res := <-resC:
					if res.Err == nil {
						atomic.AddUint64(&numReqs, 1)
						continue
					}
					if res.Err.Error() == "request manager module closed" {
						return
					}
					aErr.Store(res.Err.Error())
				case <-stopC:
					return
				}
			}
		}()
	}
	time.Sleep(testDuration)
	close(stopC)
	ts.Close()
	wg.Wait()

	if isNilErr := aErr.Load() == nil; !isNilErr {
		err := aErr.Load().(error)
		t.Errorf("unexpected error: %v", err)
	}
	num := atomic.LoadUint64(&numReqs)
	t.Logf("Mock processed requests: %v", num)
}

func TestGenReqID(t *testing.T) {
	retry := 100000
	rm := &requestManager{
		pendings: make(map[uint64]*request),
	}

	for i := 0; i != retry; i++ {
		rid := rm.genReqID()
		if _, ok := rm.pendings[rid]; ok {
			t.Errorf("rid collision")
		}
		rm.pendings[rid] = nil
	}
}

type testSuite struct {
	rm          *requestManager
	sm          *testStreamManager
	bootStreams []*testStream

	delayFunc delayFunc
	respFunc  responseFunc

	ctx    context.Context
	cancel func()
}

func newTestSuite(delayF delayFunc, respF responseFunc, numStreams int) *testSuite {
	sm := newTestStreamManager()
	rm := newRequestManager(sm)
	ctx, cancel := context.WithCancel(context.Background())

	ts := &testSuite{
		rm:          rm,
		sm:          sm,
		bootStreams: make([]*testStream, 0, numStreams),
		delayFunc:   delayF,
		respFunc:    respF,
		ctx:         ctx,
		cancel:      cancel,
	}
	for i := 0; i != numStreams; i++ {
		ts.bootStreams = append(ts.bootStreams, ts.makeTestStream(i))
	}
	return ts
}

func (ts *testSuite) Start() {
	ts.rm.Start()
	for _, st := range ts.bootStreams {
		ts.sm.addNewStream(st)
	}
}

func (ts *testSuite) Close() {
	ts.rm.Close()
	ts.cancel()
}

func (ts *testSuite) pickOneOccupiedStream() sttypes.StreamID {
	ts.rm.lock.Lock()
	defer ts.rm.lock.Unlock()

	for _, req := range ts.rm.pendings {
		return req.owner.ID()
	}
	return ""
}

type (
	// responseFunc is the function to compose a response
	responseFunc func(request *syncpb.Request) *syncpb.Response

	// delayFunc is the function to determine the delay to deliver a response
	delayFunc func() time.Duration
)

const testMsg = "test message"

func makeDefaultResponseFunc(msg string) responseFunc {
	return func(request *syncpb.Request) *syncpb.Response {
		resp := &syncpb.Response{
			ReqId: request.ReqId,
			Response: &syncpb.Response_ErrorResponse{
				ErrorResponse: &syncpb.ErrorResponse{
					Error: msg,
				},
			},
		}
		return resp
	}
}

func checkResponseMessage(resp sttypes.Response, expStr string) error {
	raw, ok := resp.GetProtobufMsg().(*syncpb.Response)
	if !ok {
		return errors.New("unexpected response type")
	}
	errResp, ok := raw.Response.(*syncpb.Response_ErrorResponse)
	if !ok {
		return errors.New("unexpected message type")
	}
	if errResp.ErrorResponse.Error != expStr {
		return fmt.Errorf("unexpected error message %v/%v", errResp.ErrorResponse.Error, expStr)
	}
	return nil
}

func makeDefaultDelayFunc(delay time.Duration) delayFunc {
	return func() time.Duration {
		return delay
	}
}

func makeOnceBlockDelayFunc(normalDelay time.Duration) delayFunc {
	// This usage of once is nasty. Avoid using once like this in production code.
	var once sync.Once
	return func() time.Duration {
		var block bool
		once.Do(func() {
			block = true
		})
		if block {
			return time.Hour
		}
		return normalDelay
	}
}

func (ts *testSuite) makeTestStream(index int) *testStream {
	stid := makeStreamID(index)
	return &testStream{
		id: stid,
		rm: ts.rm,
		deliver: func(req *syncpb.Request) {
			delay := ts.delayFunc()
			resp := ts.respFunc(req)
			go func() {
				select {
				case <-ts.ctx.Done():
				case <-time.After(delay):
					ts.rm.DeliverResponse(stid, &testResponse{resp})
				}
			}()
		},
	}
}

// normalResponseComposer is the default response composer
func normalResponseComposer(request *syncpb.Request) *syncpb.Response {
	return &syncpb.Response{
		ReqId: request.ReqId,
	}
}

type testResponse struct {
	pb *syncpb.Response
}

func (tr *testResponse) ReqID() uint64 {
	return tr.pb.ReqId
}

func (tr *testResponse) String() string {
	return "testResponse"
}

func (tr *testResponse) GetProtobufMsg() protobuf.Message {
	return tr.pb
}
