package message

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/harmony/core/types"
)

//go:generate protoc message.proto --go_out=.

// MakeGetBlocksByNumRequest makes the GetBlockByNumber request message
func MakeGetBlocksByNumRequest(rid uint64, bns []int64) *Message {
	req := &Request{
		ReqId: rid,
		Request: &Request_GetBlocksByNumRequest{
			&GetBlocksByNumRequest{
				Nums: bns,
			},
		},
	}
	return makeMessageFromRequest(req)
}

// MakeErrorResponse makes the error response
func MakeErrorResponseMessage(rid uint64, err error) *Message {
	resp := &Response{
		ReqId: rid,
		Response: &Response_ErrorResponse{
			&ErrorResponse{
				Error: err.Error(),
			},
		},
	}
	return makeMessageFromResponse(resp)
}

// MakeGetBlocksByNumResponse makes the GetBlocksByNumResponse
func MakeGetBlocksByNumResponse(rid uint64, blocks []*types.Block) (*Message, error) {
	bs := make([][]byte, 0, len(blocks))
	for _, block := range blocks {
		b, err := rlp.EncodeToBytes(block)
		if err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	resp := &Response{
		ReqId: rid,
		Response: &Response_GetBlocksByNumResponse{
			&GetBlocksByNumResponse{
				Blocks: bs,
			},
		},
	}
	return makeMessageFromResponse(resp), nil
}

func makeMessageFromRequest(req *Request) *Message {
	return &Message{
		ReqOrResp: &Message_Req{
			Req: req,
		},
	}
}

func makeMessageFromResponse(resp *Response) *Message {
	return &Message{
		ReqOrResp: &Message_Resp{
			Resp: resp,
		},
	}
}