package discovery

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	libp2p_host "github.com/libp2p/go-libp2p-core/host"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	libp2p_dis "github.com/libp2p/go-libp2p-discovery"
	libp2p_dht "github.com/libp2p/go-libp2p-kad-dht"
)

// Discovery is the interface for the underlying peer discovery protocol.
// The interface is implemented by dhtDiscovery
type Discovery interface {
	Start(ctx context.Context)
	Stop()
	Advertise(ctx context.Context, ns string) (time.Duration, error)
	FindPeers(ctx context.Context, ns string, peerLimit int) (<-chan libp2p_peer.AddrInfo, error)
}

// dhtDiscovery is a wrapper of libp2p dht discovery service. It implements Discovery
// interface.
type dhtDiscovery struct {
	dht  *libp2p_dht.IpfsDHT
	disc discovery.Discovery

	cancel func()
}

// NewDHTDiscovery creates a new dhtDiscovery that implements Discovery interface.
func NewDHTDiscovery(host libp2p_host.Host, opt DHTOption) (Discovery, error) {
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
	return &dhtDiscovery{
		dht:    dht,
		disc:   d,
		cancel: cancel,
	}, nil
}

// Start bootstrap the dht discovery service.
func (d *dhtDiscovery) Start(ctx context.Context) {
	d.dht.Bootstrap(ctx)
}

// Stop stop the dhtDiscovery service
func (d *dhtDiscovery) Stop() {
	d.cancel()
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
