package discovery

import (
	"github.com/pkg/errors"

	badger "github.com/ipfs/go-ds-badger"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	libp2p_dht "github.com/libp2p/go-libp2p-kad-dht"

	p2putils "github.com/harmony-one/harmony/p2p/utils"
)

const (
	defaultDHTBucketSize  = 10
	defaultDHTConcurrency = 5
	defaultDHTResiliency  = 2
)

// DHTOption is the configurable DHT options.
// For normal nodes, only BootNodes field need to be specified.
type DHTOption struct {
	BootNodes     p2putils.AddrList
	DataStoreFile *string // File path to store DHT data. Shall be only used for bootstrap nodes.
}

// getLibp2pRawOptions get the raw libp2p options as a slice.
func (opt DHTOption) getLibp2pRawOptions() ([]libp2p_dht.Option, error) {
	var opts []libp2p_dht.Option

	bootOption, err := getBootstrapOption(opt.BootNodes)
	if err != nil {
		return nil, err
	}
	opts = append(opts, bootOption)

	if opt.DataStoreFile != nil && len(*opt.DataStoreFile) != 0 {
		dsOption, err := getDataStoreOption(*opt.DataStoreFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, dsOption)
	}

	opts = append(opts, getDHTDefaultOptions()...)

	return opts, nil
}

func getBootstrapOption(bootNodes p2putils.AddrList) (libp2p_dht.Option, error) {
	bootPeers, err := libp2p_peer.AddrInfosFromP2pAddrs(bootNodes...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse boot node")
	}
	return libp2p_dht.BootstrapPeers(bootPeers...), nil
}

func getDataStoreOption(dataStoreFile string) (libp2p_dht.Option, error) {
	ds, err := badger.NewDatastore(dataStoreFile, nil)
	if err != nil {
		return nil, errors.Wrapf(err,
			"cannot open Badger data store at %s", dataStoreFile)
	}
	return libp2p_dht.Datastore(ds), nil
}

func getDHTDefaultOptions() []libp2p_dht.Option {
	var opts []libp2p_dht.Option

	opts = append(opts, libp2p_dht.BucketSize(defaultDHTBucketSize))
	opts = append(opts, libp2p_dht.Concurrency(defaultDHTConcurrency))
	opts = append(opts, libp2p_dht.Resiliency(defaultDHTResiliency))

	return opts
}
