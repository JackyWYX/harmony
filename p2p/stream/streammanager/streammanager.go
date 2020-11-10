package streammanager

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/harmony-one/harmony/internal/utils"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// streamManager is the implementation of StreamManager. It manages streams on
// one single protocol. It does the following job:
// 1. add a new stream to manage with when a new stream starts running.
// 2. closes a stream when some unexpected error happens.
// 3. discover new streams when number of streams is below threshold
// 4. emit stream events.
type streamManager struct {
	// streamManager only manages streams on one protocol.
	protoID sttypes.ProtoID
	config  Config
	// streams is the map of peer ID to stream
	// Note that it could happen that remote node does not share exactly the same
	// protocol ID (e.g. different version)
	streams *streamSet
	// libp2p utilities
	host host
	pf   peerFinder
	// incoming task channels
	addStreamCh chan addStreamTask
	rmStreamCh  chan rmStreamTask
	stopCh      chan stopTask
	discCh      chan discTask
	// utils
	addStreamFeed    event.Feed
	removeStreamFeed event.Feed
	logger           zerolog.Logger
	ctx              context.Context
	cancel           func()
}

// NewStreamManager creates a new stream manager
func NewStreamManager(pid sttypes.ProtoID, host host, pf peerFinder, opts ...Option) StreamManager {
	return newStreamManager(pid, host, pf, opts...)
}

// newStreamManager creates a new stream manager
func newStreamManager(pid sttypes.ProtoID, host host, pf peerFinder, opts ...Option) *streamManager {
	c := defConfig
	c.applyOptions(opts...)

	ctx, cancel := context.WithCancel(context.Background())

	logger := utils.Logger().With().Str("module", "stream manager").
		Str("protocol ID", string(pid)).Logger()

	return &streamManager{
		protoID:     pid,
		config:      c,
		streams:     newStreamSet(),
		host:        host,
		pf:          pf,
		addStreamCh: make(chan addStreamTask),
		rmStreamCh:  make(chan rmStreamTask),
		stopCh:      make(chan stopTask),
		discCh:      make(chan discTask, 1), // discCh is a buffered channel to avoid overuse of goroutine

		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the stream manager
func (sm *streamManager) Start() {
	go sm.loop()
}

// Close close the stream manager
func (sm *streamManager) Close() {
	task := stopTask{done: make(chan struct{})}
	sm.stopCh <- task

	<-task.done
}

func (sm *streamManager) loop() {
	var (
		discTicker = time.NewTicker(checkInterval)
		discCtx    context.Context
		discCancel func()
	)
	// bootstrap discovery
	sm.discCh <- discTask{}

	for {
		select {
		case <-discTicker.C:
			if !sm.softHaveEnoughStreams() {
				sm.discCh <- discTask{}
			}

		case <-sm.discCh:
			// cancel last discovery
			if discCancel != nil {
				discCancel()
			}
			discCtx, discCancel = context.WithCancel(sm.ctx)
			go func() {
				err := sm.discoverAndSetupStream(discCtx)
				if err != nil {
					sm.logger.Err(err)
				}
			}()

		case addStream := <-sm.addStreamCh:
			select {
			case <-addStream.ctx.Done():
				addStream.errC <- addStream.ctx.Err()
			default:
			}
			err := sm.handleAddStream(addStream.st)
			addStream.errC <- err

		case rmStream := <-sm.rmStreamCh:
			select {
			case <-rmStream.ctx.Done():
				rmStream.errC <- rmStream.ctx.Err()
			default:
			}
			err := sm.handleRemoveStream(rmStream.id)
			rmStream.errC <- err

		case stop := <-sm.stopCh:
			sm.cancel()
			sm.removeAllStreamOnClose()
			stop.done <- struct{}{}
			return
		}
	}
}

// NewStream handles a new stream from stream handler protocol
func (sm *streamManager) NewStream(ctx context.Context, stream sttypes.Stream) error {
	task := addStreamTask{
		ctx:  ctx,
		st:   stream,
		errC: make(chan error),
	}
	sm.addStreamCh <- task
	return <-task.errC
}

// RemoveStream close and remove a stream from stream manager
func (sm *streamManager) RemoveStream(ctx context.Context, stID sttypes.StreamID) error {
	task := rmStreamTask{
		ctx:  ctx,
		id:   stID,
		errC: make(chan error),
	}
	sm.rmStreamCh <- task
	return <-task.errC
}

type (
	addStreamTask struct {
		ctx  context.Context
		st   sttypes.Stream
		errC chan error
	}

	rmStreamTask struct {
		ctx  context.Context
		id   sttypes.StreamID
		errC chan error
	}

	discTask struct {
	}

	stopTask struct {
		done chan struct{}
	}
)

func (sm *streamManager) handleAddStream(st sttypes.Stream) error {
	id := st.ID()
	if sm.streams.size() >= sm.config.HiCap {
		return errors.New("too many streams")
	}
	if _, ok := sm.streams.get(id); ok {
		return errors.New("stream already exist")
	}

	sm.streams.addStream(st)

	sm.addStreamFeed.Send(EvtStreamAdded{st})
	return nil
}

func (sm *streamManager) handleRemoveStream(id sttypes.StreamID) error {
	st, ok := sm.streams.get(id)
	if !ok {
		return errors.New("stream not exist in manager")
	}

	sm.streams.deleteStream(id)
	// if stream number is smaller than HardLoCap, spin up the discover
	if !sm.hardHaveEnoughStream() {
		select {
		case sm.discCh <- discTask{}:
		default:
		}
	}
	sm.removeStreamFeed.Send(EvtStreamRemoved{id})
	// Hack here. We shall not add Closer interface to the Stream interface. stream manager
	// is the only module that can close the stream.
	return st.(io.Closer).Close()
}

func (sm *streamManager) removeAllStreamOnClose() {
	var wg sync.WaitGroup

	for _, st := range sm.streams.slice() {
		wg.Add(1)
		go func(st sttypes.Stream) {
			defer wg.Done()
			// Close hack here.
			err := st.(io.Closer).Close()
			if err != nil {
				sm.logger.Warn().Err(err).Str("stream ID", st.ID().String()).
					Msg("failed to close stream")
			}
		}(st)
	}
	wg.Wait()

	// Be nice. after close, the field is still accessible to prevent potential panics
	sm.streams = newStreamSet()
}

func (sm *streamManager) discoverAndSetupStream(discCtx context.Context) error {
	peers, err := sm.discover(discCtx)
	if err != nil {
		return errors.Wrap(err, "failed to discover")
	}
	for peer := range peers {
		if peer.ID == sm.host.ID() {
			continue
		}
		go func(pid libp2p_peer.ID) {
			// The ctx here is using the module context instead of discover context
			err := sm.setupStreamWithPeer(sm.ctx, pid)
			if err != nil {
				sm.logger.Warn().Err(err).Str("peerID", string(pid)).Msg("failed to setup stream with peer")
				return
			}
		}(peer.ID)
	}
	return nil
}

func (sm *streamManager) discover(ctx context.Context) (<-chan libp2p_peer.AddrInfo, error) {
	protoID := string(sm.protoID)
	discBatch := sm.config.DiscBatch
	if sm.config.HiCap-sm.streams.size() < sm.config.DiscBatch {
		discBatch = sm.config.HiCap - sm.streams.size()
	}
	if discBatch < 0 {
		return nil, nil
	}

	ctx, _ = context.WithTimeout(ctx, discTimeout)
	return sm.pf.FindPeers(ctx, protoID, discBatch)
}

func (sm *streamManager) setupStreamWithPeer(ctx context.Context, pid libp2p_peer.ID) error {
	ctx, _ = context.WithTimeout(ctx, connectTimeout)

	_, err := sm.host.NewStream(ctx, pid, protocol.ID(sm.protoID))
	return err
}

func (sm *streamManager) softHaveEnoughStreams() bool {
	return sm.streams.size() >= sm.config.SoftLoCap
}

func (sm *streamManager) hardHaveEnoughStream() bool {
	return sm.streams.size() >= sm.config.HardLoCap
}

// streamSet is the concurrency safe stream set, basically a map plus a mutex
// DO NOT ACCESS member field of this struct!!
type streamSet struct {
	streams map[sttypes.StreamID]sttypes.Stream
	lock    sync.RWMutex
}

func newStreamSet() *streamSet {
	return &streamSet{
		streams: make(map[sttypes.StreamID]sttypes.Stream),
	}
}

func (ss *streamSet) size() int {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	return len(ss.streams)
}

func (ss *streamSet) get(id sttypes.StreamID) (sttypes.Stream, bool) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	st, ok := ss.streams[id]
	return st, ok
}

func (ss *streamSet) addStream(st sttypes.Stream) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.streams[st.ID()] = st
}

func (ss *streamSet) deleteStream(id sttypes.StreamID) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	delete(ss.streams, id)
}

func (ss *streamSet) slice() []sttypes.Stream {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	sts := make([]sttypes.Stream, 0, len(ss.streams))
	for _, st := range ss.streams {
		sts = append(sts, st)
	}
	return sts
}
