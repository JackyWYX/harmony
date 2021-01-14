package requestmanager

import (
	"container/list"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestRequestQueue_pushBack(t *testing.T) {
	tests := []struct {
		initSize int
		expErr   error
	}{
		{
			initSize: 10,
			expErr:   nil,
		},
		{
			initSize: maxWaitingSize,
			expErr:   ErrQueueFull,
		},
	}
	for i, test := range tests {
		q := makeTestRequestQueue(test.initSize)
		req := wrapRequestFromRaw(makeTestRequest(uint64(test.initSize)))

		err := q.pushBack(req)
		if assErr := assertError(err, test.expErr); assErr != nil {
			t.Errorf("Test %v: %v", i, assErr)
		}
		if err != nil || test.expErr != nil {
			continue
		}

		if q.reqs.Len() != test.initSize+1 {
			t.Errorf("size unexpected: %v/%v", q.reqs.Len(), test.initSize+1)
		}
		raw, err := getTestRequestFromElem(q.reqs.Back())
		if err != nil {
			t.Error(err)
		}
		if raw.index != uint64(test.initSize) {
			t.Errorf("unexpected index at back: %v/%v", raw.index, test.initSize)
		}
	}
}

func TestRequestQueue_pushFront(t *testing.T) {
	tests := []struct {
		initSize int
		expErr   error
	}{
		{
			initSize: 10,
			expErr:   nil,
		},
		{
			initSize: maxWaitingSize,
			expErr:   ErrQueueFull,
		},
	}
	for i, test := range tests {
		q := makeTestRequestQueue(test.initSize)
		req := wrapRequestFromRaw(makeTestRequest(uint64(test.initSize)))

		err := q.pushFront(req)
		if assErr := assertError(err, test.expErr); assErr != nil {
			t.Errorf("Test %v: %v", i, assErr)
		}
		if err != nil || test.expErr != nil {
			continue
		}

		if q.reqs.Len() != test.initSize+1 {
			t.Errorf("size unexpected: %v/%v", q.reqs.Len(), test.initSize+1)
		}
		raw, err := getTestRequestFromElem(q.reqs.Front())
		if err != nil {
			t.Error(err)
		}
		if raw.index != uint64(test.initSize) {
			t.Errorf("unexpected index at back: %v/%v", raw.index, test.initSize)
		}
	}
}

func TestRequestQueue_pop(t *testing.T) {
	tests := []struct {
		q        requestQueue
		expNil   bool
		expIndex int
		expLen   int
	}{
		{
			q:      newRequestQueue(),
			expNil: true,
			expLen: 0,
		},
		{
			q:        makeTestRequestQueue(10),
			expIndex: 0,
			expLen:   9,
		},
		{
			q: func() requestQueue {
				q := newRequestQueue()
				req := wrapRequestFromRaw(makeTestRequest(10))
				q.pushBack(req)
				return q
			}(),
			expIndex: 10,
			expLen:   0,
		},
		{
			q: func() requestQueue {
				q := makeTestRequestQueue(9)
				req := wrapRequestFromRaw(makeTestRequest(10))
				q.pushFront(req)
				return q
			}(),
			expIndex: 10,
			expLen:   9,
		},
	}
	for i, test := range tests {
		req := test.q.pop()

		if test.q.reqs.Len() != test.expLen {
			t.Errorf("Test %v: unepxected size: %v / %v", i, test.q.reqs.Len(), test.expLen)
		}
		if req == nil != (test.expNil) {
			t.Errorf("test %v: unpected nil", i)
		}
		if req == nil {
			continue
		}
		index := req.Request.(*testRequest).index
		if index != uint64(test.expIndex) {
			t.Errorf("Test %v: unexpted index: %v / %v", i, index, test.expIndex)
		}
	}
}

func makeTestRequestQueue(size int) requestQueue {
	q := requestQueue{reqs: list.New()}
	for i := 0; i != size; i++ {
		q.reqs.PushBack(wrapRequestFromRaw(makeTestRequest(uint64(i))))
	}
	return q
}

func wrapRequestFromRaw(raw *testRequest) *request {
	return &request{
		Request: raw,
	}
}

func getTestRequestFromElem(elem *list.Element) (*testRequest, error) {
	req, ok := elem.Value.(*request)
	if !ok {
		return nil, errors.New("unexpected type")
	}
	raw, ok := req.Request.(*testRequest)
	if !ok {
		return nil, errors.New("unexpected raw types")
	}
	return raw, nil
}

func assertError(got, exp error) error {
	if (got == nil) != (exp == nil) {
		return fmt.Errorf("unexpected error: %v / %v", got, exp)
	}
	if got == nil {
		return nil
	}
	if !strings.Contains(got.Error(), exp.Error()) {
		return fmt.Errorf("unexpected error: %v / %v", got, exp)
	}
	return nil
}
