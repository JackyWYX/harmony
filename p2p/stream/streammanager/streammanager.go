package streammanager

import (
	"context"
	"time"

	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/harmony-one/harmony/internal/utils"
	"github.com/rs/zerolog"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/event"
	"github.com/harmony-one/harmony/p2p/discovery"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
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
	// streams is the map of peer id to stream
	// Note that it could happen that remote node does not share exactly the same
	// protocol id (e.g. different version)
	streams map[sttypes.StreamID]sttypes.Stream
	// libp2p utilities
	host host
	disc discovery.Discovery
	// incoming task channels
	addStreamCh chan addStreamTask
	rmStreamCh  chan rmStreamTask
	stopCh      chan stopTask
	discCh      chan discTask
	// utils
	event  event.Feed
	logger zerolog.Logger
	ctx    context.Context
	cancel func()
}

// TODO: change to StreamManager interface
func NewStreamManager(pid sttypes.ProtoID, host host, disc discovery.Discovery, opts ...Option) *streamManager {
	c := defConfig
	c.applyOptions(opts...)

	ctx, cancel := context.WithCancel(context.Background())

	logger := utils.Logger().With().Str("module", "stream manager").
		Str("protocol ID", string(pid)).Logger()

	return &streamManager{
		protoID:     pid,
		config:      c,
		streams:     make(map[sttypes.StreamID]sttypes.Stream),
		host:        host,
		disc:        disc,
		addStreamCh: make(chan addStreamTask),
		rmStreamCh:  make(chan rmStreamTask),
		stopCh:      make(chan stopTask),
		discCh:      make(chan discTask),

		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (sm *streamManager) Start() {
	go sm.loop()
}

func (sm *streamManager) Close() {
	task := stopTask{done: make(chan struct{})}
	sm.stopCh <- task

	<-task.done
}

func (sm *streamManager) loop() {
	var (
		t          = time.NewTicker(checkInterval)
		discCtx    context.Context
		discCancel func()
	)
	// bootstrap discovery
	sm.discCh <- discTask{}

	for {
		select {
		case <-t.C:
			if !sm.haveEnoughStreams() {
				sm.discCh <- discTask{}
			}

		case <-sm.discCh:
			// cancel last discovery
			if discCancel != nil {
				discCancel()
			}
			discCtx, discCancel = context.WithCancel(sm.ctx)
			go func() {
				err := sm.discoverAndStream(discCtx)
				if err != nil {
					sm.logger.Err(err)
				}
			}()

		case addStream := <-sm.addStreamCh:
			err := sm.handleAddStream(addStream.st)
			addStream.errC <- err

		case rmStream := <-sm.rmStreamCh:
			err := sm.handleRemoveStream(rmStream.id)
			rmStream.errC <- err

		case stop := <-sm.stopCh:
			discCancel()
			sm.cancel()
			stop.done <- struct{}{}
			return
		}
	}
}

func (sm *streamManager)

type (
	addStreamTask struct {
		st   sttypes.Stream
		errC chan error
	}

	rmStreamTask struct {
		id   sttypes.StreamID
		errC chan error
	}

	discTask struct {
	}

	stopTask struct {
		done chan struct{}
	}
)

func (sm *streamManager) haveEnoughStreams() bool {
	return len(sm.streams) >= sm.config.SoftLoCap
}

func (sm *streamManager) handleAddStream(st sttypes.Stream) error {
	id := st.ID()
	if len(sm.streams) >= sm.config.HiCap {
		return errors.New("too many streams")
	}
	if _, ok := sm.streams[id]; ok {
		return errors.New("string already exist")
	}

	sm.streams[id] = st

	sm.event.Send(EvtStreamAdded{id})
	return nil
}

func (sm *streamManager) handleRemoveStream(id sttypes.StreamID) error {
	st, ok := sm.streams[id]
	if !ok {
		return errors.New("stream not exist in manager")
	}

	delete(sm.streams, id)
	// if stream number is smaller than HardLoCap, spin up the discover
	if len(sm.streams) < sm.config.HardLoCap {
		sm.discCh <- discTask{}
	}

	return st.Close()
}

func (sm *streamManager) handleClose() {
	for _, st := range sm.streams {
		if err := st.Close(); err != nil {
			sm.logger.Warn().Str("peerID", string(st.PeerID())).Msg("cannot close stream")
		}
	}
	return
}

func (sm *streamManager) discoverAndStream(ctx context.Context) error {
	peers, err := sm.discover(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to discover")
	}
	for peer := range peers {
		go func(pid libp2p_peer.ID) {
			err := sm.setupStreamWithPeer(ctx, pid)
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
	if sm.config.HiCap-len(sm.streams) < sm.config.DiscBatch {
		discBatch = sm.config.HiCap - len(sm.streams)
	}

	ctx, cancel := context.WithTimeout(ctx, discTimeout)
	defer cancel()
	return sm.disc.FindPeers(ctx, protoID, discBatch)
}

func (sm *streamManager) setupStreamWithPeer(ctx context.Context, pid libp2p_peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	_, err := sm.host.NewStream(ctx, pid, protocol.ID(sm.protoID))
	return err
}
