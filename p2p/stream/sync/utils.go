package sync

import (
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

var (
	errUnknownReqType = errors.New("unknown request")
)

// Version returns the sync protocol version
func (p *Protocol) Version() *version.Version {
	return myVersion
}

func (p *Protocol) supportedProtoIDs() []sttypes.ProtoID {
	vs := p.supportedVersions()

	pids := make([]sttypes.ProtoID, 0, len(vs))
	for _, v := range vs {
		pids = append(pids, p.protoIDByVersion(v))
	}
	return pids
}

func (p *Protocol) supportedVersions() []*version.Version {
	return []*version.Version{version100}
}

func (p *Protocol) protoIDByVersion(v *version.Version) sttypes.ProtoID {
	spec := sttypes.ProtoSpec{
		Service:     serviceSpecifier,
		NetworkType: p.config.Network,
		ShardID:     p.config.ShardID,
		Version:     v,
	}
	return spec.ToProtoID()
}
