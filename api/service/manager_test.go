package service

import (
	"testing"

	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
)

func TestSetupServiceManager(t *testing.T) {
	m := &Manager{}
	m.SetupServiceManager()
}

func TestInit(t *testing.T) {
	if GroupIDShards[nodeconfig.ShardID(0)] != nodeconfig.NewGroupIDByShardID(0) {
		t.Errorf("GroupIDShards[0]: %v != GroupIDBeacon: %v",
			GroupIDShards[nodeconfig.ShardID(0)],
			nodeconfig.NewGroupIDByShardID(0),
		)
	}
	if len(GroupIDShards) != nodeconfig.MaxShards {
		t.Errorf("len(GroupIDShards): %v != TotalShards: %v",
			len(GroupIDShards),
			nodeconfig.MaxShards,
		)
	}
}
