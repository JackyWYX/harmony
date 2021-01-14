package sync

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	protobuf "github.com/golang/protobuf/proto"
	"github.com/harmony-one/harmony/block"
	syncpb "github.com/harmony-one/harmony/p2p/stream/protocols/sync/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/harmony-one/harmony/shard"
	"github.com/pkg/errors"
)

var (
	errUnknownReqType = errors.New("unknown request")
)

// syncResponse is the sync protocol response which implements sttypes.Response
type syncResponse struct {
	pb *syncpb.Response
}

// ReqID return the request ID of the response
func (resp *syncResponse) ReqID() uint64 {
	return resp.pb.ReqId
}

// GetProtobufMsg return the raw protobuf message
func (resp *syncResponse) GetProtobufMsg() protobuf.Message {
	return resp.pb
}

func (resp *syncResponse) String() string {
	return fmt.Sprintf("[SyncResponse %v]", resp.String())
}

// EpochStateResult is the result for GetEpochStateQuery
type EpochStateResult struct {
	header *block.Header
	state  *shard.State
}

func epochStateResultFromResponse(resp sttypes.Response) (*EpochStateResult, error) {
	sResp, ok := resp.(*syncResponse)
	if !ok || sResp == nil {
		return nil, errors.New("not sync response")
	}
	if errResp := sResp.pb.GetErrorResponse(); errResp != nil {
		return nil, errors.New(errResp.Error)
	}
	gesResp := sResp.pb.GetGetEpochStateResponse()
	if gesResp == nil {
		return nil, errors.New("not GetEpochStateResponse")
	}
	var (
		headerBytes = gesResp.HeaderBytes
		ssBytes     = gesResp.ShardState

		header *block.Header
		ss     *shard.State
	)
	if len(headerBytes) > 0 {
		if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
			return nil, err
		}
	}
	if len(ssBytes) > 0 {
		if err := rlp.DecodeBytes(ssBytes, &ss); err != nil {
			return nil, err
		}
	}
	return &EpochStateResult{
		header: header,
		state:  ss,
	}, nil
}

func (res *EpochStateResult) toMessage(rid uint64) (*syncpb.Message, error) {
	headerBytes, err := rlp.EncodeToBytes(res.header)
	if err != nil {
		return nil, err
	}
	// Shard state is not wrapped here, means no legacy shard state encoding rule as
	// in shard.EncodeWrapper.
	ssBytes, err := rlp.EncodeToBytes(res.state)
	if err != nil {
		return nil, err
	}
	return syncpb.MakeGetEpochStateResponseMessage(rid, headerBytes, ssBytes), nil
}
