package streammanager

import (
	"context"
	"errors"
	"strconv"
	"sync"

	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var _ StreamManager = &streamManager{}

var (
	myPeerID    = makePeerID(0)
	testProtoID = sttypes.ProtoID("harmony/sync/unitest/0/1.0.0")
)

func newTestStreamManager() *streamManager {
	pid := testProtoID
	host := newTestHost()
	pf := newTestPeerFinder(makeRemotePeers(100), emptyDelayFunc)

	sm := newStreamManager(pid, host, pf)
	host.sm = sm
	return sm
}

type testStream struct {
	id     sttypes.StreamID
	closed bool
}

func newTestStream(id sttypes.StreamID) *testStream {
	return &testStream{id: id}
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

func (st *testStream) Close() error {
	if st.closed {
		return errors.New("already closed")
	}
	st.closed = true
	return nil
}

type testHost struct {
	sm      *streamManager
	streams map[sttypes.StreamID]*testStream
	lock    sync.Mutex

	errHook streamErrorHook
}

type streamErrorHook func(id sttypes.StreamID, err error)

func newTestHost() *testHost {
	return &testHost{
		streams: make(map[sttypes.StreamID]*testStream),
	}
}

func (h *testHost) ID() libp2p_peer.ID {
	return myPeerID
}

// NewStream mock the upper function logic. When stream setup and running protocol, the
// upper code logic will call StreamManager to add new stream
func (h *testHost) NewStream(ctx context.Context, p libp2p_peer.ID, pids ...protocol.ID) (network.Stream, error) {
	if len(pids) == 0 {
		return nil, errors.New("nil protocol ids")
	}
	var err error
	stid := sttypes.StreamID{
		PeerID:  p2ptypes.PeerID(p),
		ProtoID: sttypes.ProtoID(pids[0]),
	}
	defer func() {
		if err != nil && h.errHook != nil {
			h.errHook(stid, err)
		}
	}()

	st := newTestStream(stid)
	h.lock.Lock()
	h.streams[stid] = st
	h.lock.Unlock()

	err = h.sm.HandleNewStream(ctx, st)
	return nil, err
}

func makePeerID(index int) libp2p_peer.ID {
	return libp2p_peer.ID(strconv.Itoa(index))
}

func makeRemotePeers(size int) []libp2p_peer.ID {
	ids := make([]libp2p_peer.ID, 0, size)
	for i := 1; i != size+1; i++ {
		ids = append(ids, makePeerID(i))
	}
	return ids
}

type testPeerFinder struct {
	peerIDs  []libp2p_peer.ID
	curIndex int
	fpHook   delayFunc
}

type delayFunc func(id libp2p_peer.ID) <-chan struct{}

func emptyDelayFunc(id libp2p_peer.ID) <-chan struct{} {
	c := make(chan struct{})
	go func() {
		c <- struct{}{}
	}()
	return c
}

func newTestPeerFinder(ids []libp2p_peer.ID, fpHook delayFunc) *testPeerFinder {
	return &testPeerFinder{
		peerIDs:  ids,
		curIndex: 0,
		fpHook:   fpHook,
	}
}

func (pf *testPeerFinder) FindPeers(ctx context.Context, ns string, peerLimit int) (<-chan libp2p_peer.AddrInfo, error) {
	if peerLimit > len(pf.peerIDs) {
		peerLimit = len(pf.peerIDs)
	}
	resC := make(chan libp2p_peer.AddrInfo)

	go func() {
		defer close(resC)

		for i := 0; i != peerLimit; i++ {
			pid := pf.peerIDs[pf.curIndex]
			select {
			case <-ctx.Done():
				return
			case <-pf.fpHook(pid):
			}
			resC <- libp2p_peer.AddrInfo{ID: pid}
			pf.curIndex++
			if pf.curIndex == len(pf.peerIDs) {
				pf.curIndex = 0
			}
		}
	}()

	return resC, nil
}