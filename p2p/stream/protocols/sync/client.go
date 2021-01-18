package sync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	protobuf "github.com/golang/protobuf/proto"
	"github.com/harmony-one/harmony/core/types"
	syncpb "github.com/harmony-one/harmony/p2p/stream/protocols/sync/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
)

// GetBlocksByNumber do getBlocksByNumberRequest through sync stream protocol.
// Return the block as result, target stream id, and error
func (p *Protocol) GetBlocksByNumber(ctx context.Context, bns []uint64, opts ...Option) ([]*types.Block, sttypes.StreamID, error) {
	req := newGetBlocksByNumberRequest(bns)

	if len(bns) > GetBlocksByNumAmountCap {
		return nil, "", fmt.Errorf("number of blocks exceed cap of %v", GetBlocksByNumAmountCap)
	}
	resp, stid, err := p.rm.DoRequest(ctx, req, opts...)
	if err != nil {
		// At this point, error can be context canceled, context timed out, or waiting queue
		// is already full.
		return nil, stid, err
	}

	// Parse and return blocks
	blocks, err := req.getBlocksFromResponse(resp)
	if err != nil {
		p.sm.RemoveStream(stid)
		return nil, stid, err
	}
	return blocks, stid, nil
}

// GetEpochState get the epoch block from querying the remote node running sync stream protocol.
// Currently, this method is only supported by beacon syncer.
// Note: use this after epoch chain is implemented.
func (p *Protocol) GetEpochState(ctx context.Context, epoch uint64, opts ...Option) (*EpochStateResult, sttypes.StreamID, error) {
	req := newGetEpochBlockRequest(epoch)

	resp, stid, err := p.rm.DoRequest(ctx, req, opts...)
	if err != nil {
		return nil, stid, err
	}

	res, err := epochStateResultFromResponse(resp)
	if err != nil {
		p.sm.RemoveStream(stid)
		return nil, stid, err
	}
	return res, stid, nil
}

// getBlocksByNumberRequest is the request for get block by numbers which implements
// sttypes.Request interface
type getBlocksByNumberRequest struct {
	bns []uint64
	msg *syncpb.Request
}

func newGetBlocksByNumberRequest(bns []uint64) *getBlocksByNumberRequest {
	msg := syncpb.MakeGetBlocksByNumRequest(bns)
	return &getBlocksByNumberRequest{
		bns: bns,
		msg: msg,
	}
}

// ReqID returns the request id of the request
func (req *getBlocksByNumberRequest) ReqID() uint64 {
	return req.msg.GetReqId()
}

// SetReqID set the request id of the request
func (req *getBlocksByNumberRequest) SetReqID(val uint64) {
	req.msg.ReqId = val
}

// String return the string representation of the request
func (req *getBlocksByNumberRequest) String() string {
	ss := make([]string, 0, len(req.bns))
	for _, bn := range req.bns {
		ss = append(ss, strconv.Itoa(int(bn)))
	}
	bnsStr := strings.Join(ss, ",")
	return fmt.Sprintf("REQUEST [GetBlockByNumber: %s]", bnsStr)
}

// IsSupportedByProto return the compatibility result with the target protoSpec
func (req *getBlocksByNumberRequest) IsSupportedByProto(target sttypes.ProtoSpec) bool {
	return target.Version.GreaterThanOrEqual(minVersion)
}

// Encode encode the request data to bytes with protobuf
func (req *getBlocksByNumberRequest) Encode() ([]byte, error) {
	msg := syncpb.MakeMessageFromRequest(req.msg)
	return protobuf.Marshal(msg)
}

func (req *getBlocksByNumberRequest) getBlocksFromResponse(resp sttypes.Response) ([]*types.Block, error) {
	sResp, ok := resp.(*syncResponse)
	if !ok || sResp == nil {
		return nil, errors.New("not sync response")
	}
	blockBytes, err := req.parseBlockBytes(sResp)
	if err != nil {
		return nil, err
	}
	blocks := make([]*types.Block, 0, len(blockBytes))
	for _, bb := range blockBytes {
		var block *types.Block
		if err := rlp.DecodeBytes(bb, &block); err != nil {
			return nil, errors.Wrap(err, "[GetBlocksByNumResponse]")
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (req *getBlocksByNumberRequest) parseBlockBytes(resp *syncResponse) ([][]byte, error) {
	if errResp := resp.pb.GetErrorResponse(); errResp != nil {
		return nil, errors.New(errResp.Error)
	}
	gbResp := resp.pb.GetGetBlocksByNumResponse()
	if gbResp == nil {
		return nil, errors.New("response not GetBlockByNumber")
	}
	return gbResp.BlocksBytes, nil
}

type getEpochBlockRequest struct {
	epoch uint64
	msg   *syncpb.Request
}

func newGetEpochBlockRequest(epoch uint64) *getEpochBlockRequest {
	msg := syncpb.MakeGetEpochStateRequest(epoch)
	return &getEpochBlockRequest{
		epoch: epoch,
		msg:   msg,
	}
}

// ReqID returns the request id of the request
func (req *getEpochBlockRequest) ReqID() uint64 {
	return req.msg.GetReqId()
}

// SetReqID set the request id of the request
func (req *getEpochBlockRequest) SetReqID(val uint64) {
	req.msg.ReqId = val
}

// String return the string representation of the request
func (req *getEpochBlockRequest) String() string {
	return fmt.Sprintf("REQUEST [GetEpochBlock: %v]", req.epoch)
}

// IsSupportedByProto return the compatibility result with the target protoSpec
func (req *getEpochBlockRequest) IsSupportedByProto(target sttypes.ProtoSpec) bool {
	return target.Version.GreaterThanOrEqual(minVersion)
}

// Encode encode the request to bytes with protobuf
func (req *getEpochBlockRequest) Encode() ([]byte, error) {
	msg := syncpb.MakeMessageFromRequest(req.msg)
	return protobuf.Marshal(msg)
}
