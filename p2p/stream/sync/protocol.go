package sync

import (
	"context"
	"time"

	"github.com/harmony-one/harmony/consensus/engine"
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/p2p/discovery"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/harmony-one/harmony/p2p/stream/utils/ratelimiter"
	"github.com/harmony-one/harmony/p2p/stream/utils/requestmanager"
	"github.com/harmony-one/harmony/p2p/stream/utils/streammanager"
	"github.com/hashicorp/go-version"
	libp2p_host "github.com/libp2p/go-libp2p-core/host"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
	"github.com/rs/zerolog"
)

const (
	// serviceSpecifier is the specifier for the service.
	serviceSpecifier = "sync"
)

var (
	version100, _ = version.NewVersion("1.0.0")

	myVersion  = version100
	minVersion = version100 // minimum version for matching function
)

type (
	// Protocol is the protocol for sync streaming
	Protocol struct {
		chain engine.ChainReader            // provide SYNC data
		rl    ratelimiter.RateLimiter       // limit the incoming request rate
		sm    streammanager.StreamManager   // stream management
		rm    requestmanager.RequestManager // deliver the response from stream
		disc  discovery.Discovery

		config Config
		logger zerolog.Logger

		ctx    context.Context
		cancel func()
		closeC chan struct{}
	}

	// Config is the sync protocol config
	Config struct {
		Chain     engine.ChainReader
		Host      libp2p_host.Host
		Discovery discovery.Discovery
		ShardID   nodeconfig.ShardID
		Network   nodeconfig.NetworkType
	}
)

// NewProtocol creates a new sync protocol
func NewProtocol(config Config) *Protocol {
	ctx, cancel := context.WithCancel(context.Background())

	sp := &Protocol{
		chain:  config.Chain,
		rl:     ratelimiter.NewRateLimiter(),
		disc:   config.Discovery,
		config: config,
		ctx:    ctx,
		cancel: cancel,
		closeC: make(chan struct{}),
	}
	sp.sm = streammanager.NewStreamManager(sp.ProtoID(), config.Host, config.Discovery)
	sp.rm = requestmanager.NewRequestManager(sp.sm)

	sp.logger = utils.Logger().With().Str("My Protocol", string(sp.ProtoID())).Logger()
	return sp
}

// Start starts the sync protocol
func (p *Protocol) Start() {
	p.sm.Start()
	p.rm.Start()
	go p.advertiseLoop()
}

func (p *Protocol) Close() {
	p.rm.Close()
	p.sm.Close()
	p.cancel()
	close(p.closeC)
}

// ProtoID return the ProtoID of the sync protocol
func (p *Protocol) ProtoID() sttypes.ProtoID {
	return p.protoIDByVersion(myVersion)
}

// Match checks the compatibility to the target protocol ID.
func (p *Protocol) Match(targetID string) bool {
	target, err := sttypes.ProtoIDToProtoSpec(sttypes.ProtoID(targetID))
	if err != nil {
		return false
	}
	if target.Service != serviceSpecifier {
		return false
	}
	if target.NetworkType != p.config.Network {
		return false
	}
	if target.ShardID != p.config.ShardID {
		return false
	}
	if target.Version.LessThan(minVersion) {
		return false
	}
	return true
}

// HandleStream is the stream handle function being registered to libp2p.
func (p *Protocol) HandleStream(raw libp2p_network.Stream) {
	st := p.wrapStream(raw)
	if err := p.sm.NewStream(st); err != nil {
		// Possibly we have reach the hard limit of the stream
		p.logger.Warn().Err(err).Str("stream ID", string(st.ID())).
			Msg("failed to add new stream")
		return
	}
	st.run()
}

func (p *Protocol) advertiseLoop() {
	for {
		sleep := p.advertise()
		select {
		case <-time.After(sleep):
		case <-p.closeC:
			return
		}
	}
}

// advertise will advertise all compatible protocol versions for helping nodes running low
// version
func (p *Protocol) advertise() time.Duration {
	var nextWait time.Duration

	pids := p.supportedProtoIDs()
	for _, pid := range pids {
		w, e := p.disc.Advertise(p.ctx, string(pid))
		if e != nil {
			p.logger.Warn().Err(e).Str("protocol", string(pid)).
				Msg("cannot advertise sync protocol")
			continue
		}
		if nextWait == 0 || nextWait > w {
			nextWait = w
		}
	}
	return nextWait
}
