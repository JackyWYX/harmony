package sync

import (
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
	"github.com/harmony-one/harmony/internal/utils"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/hashicorp/go-version"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
	"github.com/rs/zerolog"
)

const (
	// versionStr is the current version of syncing stream protocol
	versionStr = "1.0.0"

	// minVersionStr is the minimum compatible version of syncing stream protocol
	minVersionStr = "1.0.0"

	// serviceSpecifier is the specifier for the service.
	serviceSpecifier = "sync"
)

var (
	syncVersion, _ = version.NewVersion(versionStr)
	minVersion, _  = version.NewVersion(minVersionStr)
)

type (
	// Protocol is the protocol for sync streaming
	Protocol struct {
		config Config
		logger zerolog.Logger
	}

	// Config is the sync protocol config
	Config struct {
		ShardID nodeconfig.ShardID
		Network nodeconfig.NetworkType
	}
)

// NewProtocol creates a new sync protocol
func NewProtocol(config Config) *Protocol {
	sp := &Protocol{
		config: config,
	}
	sp.logger = utils.Logger().With().Str("stProto", string(sp.ProtoID())).Logger()
	return sp
}

// ProtoID return the ProtoID of the sync protocol
func (p *Protocol) ProtoID() sttypes.ProtoID {
	spec := sttypes.ProtoSpec{
		Service:     serviceSpecifier,
		NetworkType: p.config.Network,
		ShardID:     p.config.ShardID,
		Version:     syncVersion,
	}
	return spec.ToProtoID()
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

// TODO: implement this
func (p *Protocol) HandleStream(st libp2p_network.Stream) {

}

// Version returns the sync protocol version
func (p *Protocol) Version() *version.Version {
	return syncVersion
}
