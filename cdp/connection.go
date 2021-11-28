package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/gorilla/websocket"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
)

func connect(ctx context.Context, websocketURL string) (*connection, error) {
	wd := &websocket.Dialer{
		HandshakeTimeout: time.Second * 10,
		ReadBufferSize:   1 << 20,                   // why?
		WriteBufferSize:  1 << 20,                   // why?
		Proxy:            http.ProxyFromEnvironment, // necessary?
	}
	conn, _, err := wd.DialContext(ctx, websocketURL, http.Header{})
	if err != nil {
		err = fmt.Errorf("connect: %w", err)
	}
	return &connection{
		ws: conn,
	}, err
}

type connection struct {
	ws  *websocket.Conn
	mid int64 // cdp message ID
}

func (c *connection) recv(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("connection.recv:ctx.Done: %w", ctx.Err())
	default:
	}
	_, buf, err := c.ws.ReadMessage()
	if err != nil {
		err = fmt.Errorf("connection.recv:ws.ReadMessage: %w", err)
	}
	return buf, err
}

func (c *connection) send(ctx context.Context, buf []byte) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("connection.send:ctx.Done: %w", ctx.Err())
	default:
	}
	w, err := c.ws.NextWriter(websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("connection.write:w.NextWriter: %w", err)
	}
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("connection.write:w.Write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("connection.write:w.Close: %w", err)
	}
	return nil
}

func (c *connection) sendCDPMsg(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	var (
		buf []byte
		err error
	)
	if params != nil {
		buf, err = easyjson.Marshal(params)
	}
	if err != nil {
		return fmt.Errorf("connection:Execute:Marshal: %w", err)
	}

	msg := &cdproto.Message{
		ID:     atomic.AddInt64(&c.mid, 1), // cdp message ID
		Method: cdproto.MethodType(method),
		Params: buf,
	}

	var encoder jwriter.Writer
	msg.MarshalEasyJSON(&encoder)
	if err := encoder.Error; err != nil {
		return fmt.Errorf("connection:Execute:encoder:MarshalEasyJSON: %w", err)
	}
	// what's difference of directly passing `buf` here?
	// instead of calling this `BuildBytes`?
	if buf, err = encoder.BuildBytes(); err != nil {
		return fmt.Errorf("connection:Execute:encoder.BuildBytes: %w", err)
	}

	return c.send(ctx, buf)
}

func (c *connection) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	return c.sendCDPMsg(ctx, method, params, res)
}
