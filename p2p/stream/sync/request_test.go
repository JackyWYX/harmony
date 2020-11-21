package sync

import sttypes "github.com/harmony-one/harmony/p2p/stream/types"

var _ sttypes.Request = &getBlocksByNumberRequest{}
