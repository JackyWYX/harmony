package requestmanager

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

// stream is the wrapped version of sttypes.Stream.
// TODO: enable stream handle multiple requests
type stream struct {
	sttypes.Stream
	req *request // currently one stream is dealing with one request
}

// request is the wrapped request within module
type request struct {
	sttypes.Request // underlying request
	// result field
	respC chan *message.Response // channel to receive response from delivered message
	err   error
	// concurrency control
	waitCh chan struct{} // channel to wait for the request to be canceled or answered
	// stream info
	owner *stream // Current owner
	// utils
	lock sync.RWMutex
}

func (req *request) ReqID() uint64 {
	req.lock.RLock()
	defer req.lock.RUnlock()

	return req.Request.ReqID()
}

func (req *request) SetReqID(val uint64) {
	req.lock.Lock()
	defer req.lock.Unlock()

	req.Request.SetReqID(val)
}

func (st *stream) clearPendingRequest() *request {
	req := st.req
	if req == nil {
		return nil
	}
	req.owner = nil
	st.req = nil
	return req
}

type deliverData struct {
	resp *message.Response
	stID sttypes.StreamID
}

// response is the response for stream requests
type response struct {
	resp *message.Response
	err  error
}

// requestQueue is a wrapper of double linked list with Request as type
type requestQueue struct {
	reqs *list.List
	lock sync.Mutex
}

func (q *requestQueue) pushBack(req *request) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.reqs.Len() > maxWaitingSize {
		return fmt.Errorf("waiting request queue size exceeding %v", maxWaitingSize)
	}
	q.reqs.PushBack(req)
	return nil
}

func (q *requestQueue) pushFront(req *request) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.reqs.Len() > maxWaitingSize {
		return fmt.Errorf("waiting request queue size exceeding %v", maxWaitingSize)
	}
	q.reqs.PushFront(req)
	return nil
}

// Note: pop might return nil
func (q *requestQueue) pop() *request {
	q.lock.Lock()
	defer q.lock.Unlock()

	elem := q.reqs.Front()
	if elem == nil {
		return nil
	}
	q.reqs.Remove(elem)
	return elem.Value.(*request)
}
