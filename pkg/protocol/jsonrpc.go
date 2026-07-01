package protocol

import (
	"encoding/json"
	"fmt"
)

const Version = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

func NewRequest(id, method string, params any) (Request, error) {
	req := Request{
		JSONRPC: Version,
		ID:      id,
		Method:  method,
	}
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return Request{}, fmt.Errorf("marshal params: %w", err)
		}
		req.Params = data
	}
	return req, nil
}

func NewResponse(id string, result any) (Response, error) {
	resp := Response{
		JSONRPC: Version,
		ID:      id,
	}
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return Response{}, fmt.Errorf("marshal result: %w", err)
		}
		resp.Result = data
	}
	return resp, nil
}

func NewError(id string, code int, message string) Response {
	return Response{
		JSONRPC: Version,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}
