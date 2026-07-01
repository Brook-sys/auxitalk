package rpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/protocol"
)

func TestReadRequest(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"1","method":"test.method","params":{"foo":"bar"}}` + "\n"
	codec := NewCodec(strings.NewReader(input), nil, 1024)

	req, err := codec.ReadRequest(context.Background())
	if err != nil {
		t.Fatalf("read request: %v", err)
	}

	if req.ID != "1" {
		t.Errorf("expected id 1, got %q", req.ID)
	}
	if req.Method != "test.method" {
		t.Errorf("expected test.method, got %q", req.Method)
	}
}

func TestReadResponse(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"1","result":{"ok":true}}` + "\n"
	codec := NewCodec(strings.NewReader(input), nil, 1024)

	resp, err := codec.ReadResponse(context.Background())
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if resp.ID != "1" {
		t.Errorf("expected id 1, got %q", resp.ID)
	}
}

func TestWriteRequest(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(nil, &buf, 1024)

	req, _ := protocol.NewRequest("1", "test.method", nil)
	if err := codec.WriteRequest(context.Background(), req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":"1","method":"test.method"}` + "\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestWriteResponse(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(nil, &buf, 1024)

	resp, _ := protocol.NewResponse("1", nil)
	if err := codec.WriteResponse(context.Background(), resp); err != nil {
		t.Fatalf("write response: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":"1"}` + "\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestReadPayloadTooLarge(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"1","method":"test","params":"` + strings.Repeat("x", 100) + `"}` + "\n"
	codec := NewCodec(strings.NewReader(input), nil, 50)

	_, err := codec.ReadRequest(context.Background())
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestWritePayloadTooLarge(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(nil, &buf, 50)

	req, _ := protocol.NewRequest("1", "test", strings.Repeat("x", 100))
	err := codec.WriteRequest(context.Background(), req)
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestReadContextTimeout(t *testing.T) {
	r, _ := io.Pipe() // pipe reads block forever until written or closed
	codec := NewCodec(r, nil, 1024)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := codec.ReadRequest(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}
