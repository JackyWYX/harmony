package sttypes

// TODO: test this file

import (
	"fmt"

	"github.com/hashicorp/go-version"
	libp2p_proto "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"

	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
)

const (
	// ProtoIDCommonPrefix is the common prefix for stream protocol
	ProtoIDCommonPrefix = "harmony"

	// ProtoIDFormat is the format of stream protocol ID
	ProtoIDFormat = "%s/%s/%s/%d/%s"
)

// ProtoID is the protocol id for streaming, an alias of libp2p stream protocol ID。
// The stream protocol ID is composed of following components:
// 1. Service - Currently, only sync service is supported.
// 2. NetworkType - mainnet, testnet, stn, e.t.c.
// 3. ShardID - shard ID of the current protocol.
// 4. Version - Stream protocol version for backward compatibility.
type ProtoID libp2p_proto.ID

// ProtoSpec is the un-serialized stream proto id specification
type ProtoSpec struct {
	Service     string
	NetworkType nodeconfig.NetworkType
	ShardID     nodeconfig.ShardID
	Version     *version.Version
}

// ToProtoID convert a ProtoSpec to ProtoID.
func (spec ProtoSpec) ToProtoID() ProtoID {
	s := fmt.Sprintf(ProtoIDFormat, ProtoIDCommonPrefix, spec.Service,
		spec.NetworkType, spec.ShardID, spec.Version.String())
	return ProtoID(s)
}

// ProtoIDToProtoSpec converts a ProtoID to ProtoSpec
func ProtoIDToProtoSpec(id ProtoID) (ProtoSpec, error) {
	var (
		prefix, service, versionStr string
		networkType                 nodeconfig.NetworkType
		shardID                     nodeconfig.ShardID
	)
	_, err := fmt.Sscanf(string(id), ProtoIDFormat, prefix, service, networkType, shardID, versionStr)
	if err != nil {
		return ProtoSpec{}, errors.Wrap(err, "failed to parse ProtoID")
	}
	if prefix != ProtoIDCommonPrefix {
		return ProtoSpec{}, errors.New("unexpected prefix")
	}
	version, err := version.NewVersion(versionStr)
	if err != nil {
		return ProtoSpec{}, errors.Wrap(err, "unexpected version string")
	}
	return ProtoSpec{
		Service:     service,
		NetworkType: networkType,
		ShardID:     shardID,
		Version:     version,
	}, nil
}
