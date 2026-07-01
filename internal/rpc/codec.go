package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/Brook-sys/auxitalk/pkg/protocol"
)

var ErrPayloadTooLarge = errors.New("jsonrpc payload too large")

type Codec struct {
	reader         *bufio.Reader
	writer         io.Writer
	writeMu        sync.Mutex
	maxPayloadSize int64
}

func NewCodec(reader io.Reader, writer io.Writer, maxPayloadSize int64) *Codec {
	if maxPayloadSize <= 0 {
		maxPayloadSize = 1024 * 1024
	}
	return &Codec{
		reader:         bufio.NewReader(reader),
		writer:         writer,
		maxPayloadSize: maxPayloadSize,
	}
}

func (c *Codec) ReadRequest(ctx context.Context) (protocol.Request, error) {
	data, err := c.readLine(ctx)
	if err != nil {
		return protocol.Request{}, err
	}

	var request protocol.Request
	if err := json.Unmarshal(data, &request); err != nil {
		return protocol.Request{}, fmt.Errorf("parse jsonrpc request: %w", err)
	}
	if request.JSONRPC != protocol.Version {
		return protocol.Request{}, errors.New("invalid jsonrpc version")
	}
	if request.Method == "" {
		return protocol.Request{}, errors.New("jsonrpc method is required")
	}

	return request, nil
}

func (c *Codec) ReadResponse(ctx context.Context) (protocol.Response, error) {
	data, err := c.readLine(ctx)
	if err != nil {
		return protocol.Response{}, err
	}

	var response protocol.Response
	if err := json.Unmarshal(data, &response); err != nil {
		return protocol.Response{}, fmt.Errorf("parse jsonrpc response: %w", err)
	}
	if response.JSONRPC != protocol.Version {
		return protocol.Response{}, errors.New("invalid jsonrpc version")
	}
	if response.ID == "" {
		return protocol.Response{}, errors.New("jsonrpc response id is required")
	}

	return response, nil
}

func (c *Codec) WriteRequest(ctx context.Context, request protocol.Request) error {
	if request.JSONRPC == "" {
		request.JSONRPC = protocol.Version
	}
	return c.writeJSON(ctx, request)
}

func (c *Codec) WriteResponse(ctx context.Context, response protocol.Response) error {
	if response.JSONRPC == "" {
		response.JSONRPC = protocol.Version
	}
	return c.writeJSON(ctx, response)
}

func (c *Codec) readLine(ctx context.Context) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	resultCh := make(chan result, 1)
	go func() {
		data, err := c.reader.ReadBytes('\n')
		resultCh <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		if int64(len(result.data)) > c.maxPayloadSize {
			return nil, ErrPayloadTooLarge
		}
		return result.data, nil
	}
}

func (c *Codec) writeJSON(ctx context.Context, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal jsonrpc message: %w", err)
	}
	if int64(len(data)+1) > c.maxPayloadSize {
		return ErrPayloadTooLarge
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := c.writer.Write(append(data, '\n'))
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
