package sync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	"github.com/pkg/errors"
)

func (p *Protocol) GetBlocksByNumber(ctx context.Context, bns []uint64) ([]*types.Block, error) {
	req := newGetBlocksByNumberRequest(bns)

	resp, stid, err := p.rm.DoRequest(ctx, req)
	if err != nil {
		// At this point, error can be context canceled, context timed out, or waiting queue
		// is already full.
		return nil, err
	}

	// Parse and return blocks
	blocks, err := req.getBlocksFromResponse(resp)
	if err != nil {
		p.sm.RemoveStream(stid)
		return nil, err
	}
	if err := req.validateBlocks(blocks); err != nil {
		p.sm.RemoveStream(stid)
		return nil, err
	}
	return blocks, nil
}

// getBlocksByNumberRequest is the request for get block by numbers which implements
// Request interface
type getBlocksByNumberRequest struct {
	bns []uint64
	msg *message.Request
}

func newGetBlocksByNumberRequest(bns []uint64) *getBlocksByNumberRequest {
	msg := message.MakeGetBlocksByNumRequest(bns)
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

// GetRequestMessage get the raw protobuf message ready for send
func (req *getBlocksByNumberRequest) GetRequestMessage() *message.Request {
	return req.msg
}

func (req *getBlocksByNumberRequest) getBlocksFromResponse(resp *message.Response) ([]*types.Block, error) {
	if errResp := resp.GetErrorResponse(); errResp != nil {
		return nil, errors.New(errResp.Error)
	}
	gbResp := resp.GetGetBlocksByNumResponse()
	if gbResp == nil {
		return nil, fmt.Errorf("[GetBlocksByNumResponse] unexpected response type: %v", gbResp)
	}
	blocks := make([]*types.Block, 0, len(gbResp.Blocks))
	for _, bb := range gbResp.Blocks {
		var block *types.Block
		if err := rlp.DecodeBytes(bb, &block); err != nil {
			return nil, errors.Wrap(err, "[GetBlocksByNumResponse]")
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// validateBlocks validates whether the delivered block is expected from requested block number
// The signature is not validated here.
func (req *getBlocksByNumberRequest) validateBlocks(blocks []*types.Block) error {
	for i, block := range blocks {
		if block != nil && block.NumberU64() != req.bns[i] {
			return fmt.Errorf("[GetBlocksByNumResponse] unexpected block number: %v/%v",
				block.NumberU64(), req.bns[i])
		}
	}
	return nil
}
