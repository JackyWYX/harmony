package message

// MakeGetBlocksByNumRequest makes the GetBlockByNumber request
func MakeGetBlocksByNumRequest(bns []uint64) *Request {
	return &Request{
		Request: &Request_GetBlocksByNumRequest{
			GetBlocksByNumRequest: &GetBlocksByNumRequest{
				Nums: bns,
			},
		},
	}
}

// MakeGetEpochStateRequest make GetEpochBlock request
func MakeGetEpochStateRequest(epoch uint64) *Request {
	return &Request{
		Request: &Request_GetEpochStateRequest{
			GetEpochStateRequest: &GetEpochStateRequest{
				Epoch: epoch,
			},
		},
	}
}

// MakeErrorResponse makes the error response
func MakeErrorResponseMessage(rid uint64, err error) *Message {
	resp := MakeErrorResponse(rid, err)
	return makeMessageFromResponse(resp)
}

// MakeErrorResponse makes the error response as a response
func MakeErrorResponse(rid uint64, err error) *Response {
	return &Response{
		ReqId: rid,
		Response: &Response_ErrorResponse{
			&ErrorResponse{
				Error: err.Error(),
			},
		},
	}
}

// MakeGetBlocksByNumResponseMessage makes the GetBlocksByNumResponse of Message type
func MakeGetBlocksByNumResponseMessage(rid uint64, blocksBytes [][]byte) (*Message, error) {
	resp, err := MakeGetBlocksByNumResponse(rid, blocksBytes)
	if err != nil {
		return nil, err
	}
	return makeMessageFromResponse(resp), nil
}

// MakeGetBlocksByNumResponseMessage make the GetBlocksByNumResponse of Response type
func MakeGetBlocksByNumResponse(rid uint64, blocksBytes [][]byte) (*Response, error) {
	return &Response{
		ReqId: rid,
		Response: &Response_GetBlocksByNumResponse{
			GetBlocksByNumResponse: &GetBlocksByNumResponse{
				BlocksBytes: blocksBytes,
			},
		},
	}, nil
}

// MakeGetEpochStateResponse makes GetEpochStateResponse as message
func MakeGetEpochStateResponseMessage(rid uint64, headerBytes []byte, ssBytes []byte) *Message {
	resp := MakeGetEpochStateResponse(rid, headerBytes, ssBytes)
	return makeMessageFromResponse(resp)
}

// MakeEpochStateResponse makes GetEpochStateResponse as response
func MakeGetEpochStateResponse(rid uint64, headerBytes []byte, ssBytes []byte) *Response {
	return &Response{
		ReqId: rid,
		Response: &Response_GetEpochStateResponse{
			GetEpochStateResponse: &GetEpochStateResponse{
				HeaderBytes: headerBytes,
				ShardState:  ssBytes,
			},
		},
	}
}

// MakeMessageFromRequest makes a message from the request
func MakeMessageFromRequest(req *Request) *Message {
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
