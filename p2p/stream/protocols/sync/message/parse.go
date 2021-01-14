package message

import (
	"fmt"

	"github.com/pkg/errors"
)

// ResponseError is the error from an error response
type ResponseError struct {
	msg string
}

// Error is the error string of ResponseError
func (err *ResponseError) Error() string {
	return fmt.Sprintf("[RESPONSE] %v", err.msg)
}

// GetBlocksByNumberResponse parse the message to GetBlocksByNumberResponse
func (msg *Message) GetBlocksByNumberResponse() (*GetBlocksByNumResponse, error) {
	resp := msg.GetResp()
	if resp == nil {
		return nil, errors.New("not response message")
	}
	if errResp := resp.GetErrorResponse(); errResp != nil {
		return nil, &ResponseError{errResp.Error}
	}
	gbResp := resp.GetGetBlocksByNumResponse()
	if gbResp == nil {
		return nil, errors.New("not GetBlocksByNumResponse")
	}
	return gbResp, nil
}

// GetEpochStateResponse parse the message to GetEpochStateResponse
func (msg *Message) GetEpochStateResponse() (*GetEpochStateResponse, error) {
	resp := msg.GetResp()
	if resp == nil {
		return nil, errors.New("not response message")
	}
	if errResp := resp.GetErrorResponse(); errResp != nil {
		return nil, &ResponseError{errResp.Error}
	}
	gesResp := resp.GetGetEpochStateResponse()
	if gesResp == nil {
		return nil, errors.New("not GetEpochStateResponse")
	}
	return gesResp, nil
}
