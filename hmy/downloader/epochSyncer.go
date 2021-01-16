package downloader

import "github.com/harmony-one/harmony/p2p/stream/protocols/sync"

type epochSyncer struct {
	syncProtocol *sync.Protocol
}
