package sync

import (
	"github.com/harmony-one/harmony/p2p/stream/sync/syncpb"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
)

var (
	_ sttypes.Request  = &getBlocksByNumberRequest{}
	_ sttypes.Response = &syncResponse{&syncpb.Response{}}
)
