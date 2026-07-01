package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req, err := NewRequest("1", "test.method", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected 2.0, got %q", req.JSONRPC)
	}
	if req.ID != "1" {
		t.Errorf("expected 1, got %q", req.ID)
	}
	if req.Method != "test.method" {
		t.Errorf("expected test.method, got %q", req.Method)
	}

	var params map[string]string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params["foo"] != "bar" {
		t.Errorf("unexpected params: %v", params)
	}
}

func TestNewResponse(t *testing.T) {
	resp, err := NewResponse("1", map[string]string{"ok": "yes"})
	if err != nil {
		t.Fatalf("new response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected 2.0, got %q", resp.JSONRPC)
	}
	if resp.ID != "1" {
		t.Errorf("expected 1, got %q", resp.ID)
	}
	if resp.Error != nil {
		t.Fatal("expected no error")
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["ok"] != "yes" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestNewError(t *testing.T) {
	resp := NewError("1", ErrMethodNotFound, "method not found")

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected 2.0, got %q", resp.JSONRPC)
	}
	if resp.ID != "1" {
		t.Errorf("expected 1, got %q", resp.ID)
	}
	if resp.Result != nil {
		t.Fatal("expected no result")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != ErrMethodNotFound {
		t.Errorf("expected code %d, got %d", ErrMethodNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "method not found" {
		t.Errorf("expected method not found, got %q", resp.Error.Message)
	}

	expectedStr := "jsonrpc error -32601: method not found"
	if resp.Error.Error() != expectedStr {
		t.Errorf("expected string %q, got %q", expectedStr, resp.Error.Error())
	}
}
