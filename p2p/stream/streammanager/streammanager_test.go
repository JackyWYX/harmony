package streammanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
)

const (
	defTestWait = 100 * time.Millisecond
)

// When started, discover will be run at bootstrap
func TestStreamManager_BootstrapDisc(t *testing.T) {
	sm := newTestStreamManager()
	sm.host.(*testHost).errHook = func(id sttypes.StreamID, err error) {
		t.Errorf("%s stream error: %v", id, err)
	}

	// After bootstrap, stream manager shall discover streams and setup connection
	// Note host will mock the upper code logic to call sm.NewStream in this case
	sm.Start()
	time.Sleep(defTestWait)
	if gotSize := sm.streams.size(); gotSize != sm.config.DiscBatch {
		t.Errorf("unexpected stream size: %v != %v", gotSize, sm.config.DiscBatch)
	}
}

// After close, all stream shall be closed and removed
func TestStreamManager_Close(t *testing.T) {
	sm := newTestStreamManager()
	// Bootstrap
	sm.Start()
	time.Sleep(defTestWait)
	// Close stream manager, all stream shall be closed and removed
	closeDone := make(chan struct{})
	go func() {
		sm.Close()
		closeDone <- struct{}{}
	}()
	select {
	case <-time.After(defTestWait):
		t.Errorf("still not closed")
	case <-closeDone:
	}
	// Check stream been removed from stream manager and all streams to be closed
	if sm.streams.size() != 0 {
		t.Errorf("after close, stream not removed from stream manager")
	}
	host := sm.host.(*testHost)
	for _, st := range host.streams {
		if !st.closed {
			t.Errorf("after close, stream still not closed")
		}
	}
}

// Close shall terminate the current discover at once
func TestStreamManager_CloseDisc(t *testing.T) {
	sm := newTestStreamManager()
	// discover will be blocked forever
	sm.pf.(*testPeerFinder).fpHook = func(id libp2p_peer.ID) <-chan struct{} {
		select {}
	}
	sm.Start()
	time.Sleep(defTestWait)
	// Close stream manager, all stream shall be closed and removed
	closeDone := make(chan struct{})
	go func() {
		sm.Close()
		closeDone <- struct{}{}
	}()
	select {
	case <-time.After(defTestWait):
		t.Errorf("close shall unblock the current discovery")
	case <-closeDone:
	}
}

// Each time discTicker ticks, it will cancel last discovery, and start a new one
func TestStreamManager_refreshDisc(t *testing.T) {
	sm := newTestStreamManager()
	// discover will be blocked for the first time but good for second time
	var once sync.Once
	sm.pf.(*testPeerFinder).fpHook = func(id libp2p_peer.ID) <-chan struct{} {
		var sendSig = true
		once.Do(func() {
			sendSig = false
		})
		c := make(chan struct{}, 1)
		if sendSig {
			c <- struct{}{}
		}
		return c
	}
	sm.Start()
	time.Sleep(defTestWait)

	sm.discCh <- struct{}{}
	time.Sleep(defTestWait)

	// We shall now have non-zero streams setup
	if sm.streams.size() == 0 {
		t.Errorf("stream size still zero after refresh")
	}
}

func TestStreamManager_HandleNewStream(t *testing.T) {
	tests := []struct {
		stream  sttypes.Stream
		expSize int
		expErr  error
	}{
		{
			stream: newTestStream(sttypes.StreamID{
				PeerID:  p2ptypes.PeerID(makePeerID(100)),
				ProtoID: testProtoID,
			}),
			expSize: defDiscBatch + 1,
			expErr:  nil,
		},
		{
			stream: newTestStream(sttypes.StreamID{
				PeerID:  p2ptypes.PeerID(makePeerID(1)),
				ProtoID: testProtoID,
			}),
			expSize: defDiscBatch,
			expErr:  errors.New("stream already exist"),
		},
	}
	for i, test := range tests {
		sm := newTestStreamManager()
		sm.Start()
		time.Sleep(defTestWait)

		err := sm.NewStream(context.Background(), test.stream)
		if assErr := assertError(err, test.expErr); assErr != nil {
			t.Errorf("Test %v: %v", i, assErr)
		}

		if sm.streams.size() != test.expSize {
			t.Errorf("Test %v: unexpected stream size: %v / %v", i, sm.streams.size(),
				test.expSize)
		}
	}
}

func TestStreamManager_HandleRemoveStream(t *testing.T) {
	tests := []struct {
		id      sttypes.StreamID
		expSize int
		expErr  error
	}{
		{
			id: sttypes.StreamID{
				PeerID:  p2ptypes.PeerID(makePeerID(1)),
				ProtoID: testProtoID,
			},
			expSize: defDiscBatch - 1,
			expErr:  nil,
		},
		{
			id: sttypes.StreamID{
				PeerID:  p2ptypes.PeerID(makePeerID(100)),
				ProtoID: testProtoID,
			},
			expSize: defDiscBatch,
			expErr:  errors.New("stream not exist in manager"),
		},
	}
	for i, test := range tests {
		sm := newTestStreamManager()
		sm.Start()
		time.Sleep(defTestWait)

		err := sm.RemoveStream(context.Background(), test.id)
		if assErr := assertError(err, test.expErr); assErr != nil {
			t.Errorf("Test %v: %v", i, assErr)
		}

		if sm.streams.size() != test.expSize {
			t.Errorf("Test %v: unexpected stream size: %v / %v", i, sm.streams.size(),
				test.expSize)
		}
	}
}

// When number of streams is smaller than hard low limit, discover will be triggered
func TestStreamManager_HandleRemoveStream_Disc(t *testing.T) {
	sm := newTestStreamManager()
	sm.Start()
	time.Sleep(defTestWait)

	// Remove DiscBatch - HardLoCap + 1 streams
	num := 0
	for _, st := range sm.streams.slice() {
		if err := sm.RemoveStream(context.Background(), st.ID()); err != nil {
			t.Error(err)
		}
		num++
		if num == sm.config.DiscBatch-sm.config.HardLoCap+1 {
			break
		}
	}

	// Last remove stream will also trigger discover
	time.Sleep(defTestWait)
	if sm.streams.size() != sm.config.HardLoCap+sm.config.DiscBatch-1 {
		t.Errorf("unexpected stream number %v / %v", sm.streams.size(), sm.config.HardLoCap+sm.config.DiscBatch-1)
	}

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
