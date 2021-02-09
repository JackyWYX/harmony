package discovery

import (
	"context"
	"math/rand"
	"time"

	"github.com/harmony-one/harmony/internal/utils"
	"github.com/libp2p/go-libp2p-core/discovery"
	libp2p_host "github.com/libp2p/go-libp2p-core/host"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	libp2p_dis "github.com/libp2p/go-libp2p-discovery"
	libp2p_dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Discovery is the interface for the underlying peer discovery protocol.
// The interface is implemented by dhtDiscovery
type Discovery interface {
	Start() error
	Close() error
	Advertise(ctx context.Context, ns string) (time.Duration, error)
	FindPeers(ctx context.Context, ns string, peerLimit int) (<-chan libp2p_peer.AddrInfo, error)
	GetRawDiscovery() discovery.Discovery
}

// dhtDiscovery is a wrapper of libp2p dht discovery service. It implements Discovery
// interface.
type dhtDiscovery struct {
	dht  *libp2p_dht.IpfsDHT
	disc discovery.Discovery
	host libp2p_host.Host

	opt    DHTConfig
	logger zerolog.Logger
	ctx    context.Context
	cancel func()
}

// NewDHTDiscovery creates a new dhtDiscovery that implements Discovery interface.
func NewDHTDiscovery(host libp2p_host.Host, opt DHTConfig) (Discovery, error) {
	opts, err := opt.getLibp2pRawOptions()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	dht, err := libp2p_dht.New(ctx, host, opts...)
	if err != nil {
		return nil, err
	}
	d := libp2p_dis.NewRoutingDiscovery(dht)

	logger := utils.Logger().With().Str("module", "discovery").Logger()
	return &dhtDiscovery{
		dht:    dht,
		disc:   d,
		host:   host,
		opt:    opt,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start bootstrap the dht discovery service.
func (d *dhtDiscovery) Start() error {
	if err := d.connectBootStrapPeers(); err != nil {
		return err
	}
	return d.dht.Bootstrap(d.ctx)
}

// Stop stop the dhtDiscovery service
func (d *dhtDiscovery) Close() error {
	d.cancel()
	return nil
}

// Advertise advertises a service
func (d *dhtDiscovery) Advertise(ctx context.Context, ns string) (time.Duration, error) {
	return d.disc.Advertise(ctx, ns)
}

// FindPeers discovers peers providing a service
func (d *dhtDiscovery) FindPeers(ctx context.Context, ns string, peerLimit int) (<-chan libp2p_peer.AddrInfo, error) {
	opt := discovery.Limit(peerLimit)
	return d.disc.FindPeers(ctx, ns, opt)
}

// GetRawDiscovery get the raw discovery to be used for libp2p pubsub options
func (d *dhtDiscovery) GetRawDiscovery() discovery.Discovery {
	return d.disc
}

// connectBootStrapPeers connect to bootstrap peers.
// Currently, specifying bootstrap nodes in dht options does not connect to boot nodes
// immediately (only after 2 min initial wait in fixLowPeersRoutine). Thus we need to
// connect to bootstrap peers in this module. Hopefully this logic will be implemented
// in libp2p naturally.
func (d *dhtDiscovery) connectBootStrapPeers() error {
	var found int

	for _, i := range rand.Perm(len(d.opt.BootNodes)) {
		bn := d.opt.BootNodes[i]
		if err := d.host.Connect(d.ctx, bn); err == nil {
			found++
		} else {
			d.logger.Warn().Err(err).Msg("failed connect boot node")
		}
		if found == maxNBoostrappers {
			break
		}
	}

	if found == 0 {
		return errors.New("failed to connect to any bootstrap nodes")
	}
	return nil
}
