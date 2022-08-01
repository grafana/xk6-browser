package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	cdpext "github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/grafana/xk6-browser/cdp/domains"
	"github.com/grafana/xk6-browser/log"
	"github.com/mailru/easyjson"
)

var _ cdp.Executor = &Client{}

// Client manages CDP communication with the browser.
type Client struct {
	ctx    context.Context
	logger *log.Logger

	Browser domains.Browser
	Page    domains.Page
	Target  domains.Target

	conn      *connection
	msgID     int64
	recvCh    chan *cdproto.Message
	sendCh    chan *cdproto.Message
	msgSubsMu sync.RWMutex
	msgSubs   map[int64]chan *cdproto.Message
	// closeCh chan int
	errorCh chan error
	done    chan struct{}

	sessionsMu sync.RWMutex
	sessions   map[target.SessionID]*session
	watcher    *eventWatcher
	wsURL      string
}

// NewClient returns a new Client that is unusable until a CDP connection is
// established with Connect().
func NewClient(ctx context.Context, logger *log.Logger) *Client {
	c := &Client{
		ctx:    ctx,
		logger: logger,
		recvCh: make(chan *cdproto.Message),
		sendCh: make(chan *cdproto.Message, 32), // Buffered to avoid blocking in Execute
		// msgID:   1000,
		msgSubs: make(map[int64]chan *cdproto.Message),
		watcher: newEventWatcher(ctx),
	}

	// TODO: Extract Execute outside of Client?
	c.Page = domains.NewPage(c)
	c.Target = domains.NewTarget(c)
	c.Browser = domains.NewBrowser(c)

	return c
}

// Connect to the browser that exposes a CDP API at wsURL.
func (c *Client) Connect(wsURL string) (err error) {
	if c.wsURL != "" {
		return fmt.Errorf("CDP connection already established to %q", c.wsURL)
	}

	if c.conn, err = newConnection(c.ctx, wsURL, c.logger); err != nil {
		return
	}
	c.logger.Infof("cdp", "established CDP connection to %q", wsURL)
	c.wsURL = wsURL

	go c.recvLoop()
	go c.recvMsgLoop()
	go c.sendLoop()

	return nil
}

// Disconnect from the browser's CDP API.
func (c *Client) Disconnect() {
	c.conn.Close()
}

// Execute implements cdproto.Executor and performs a synchronous send and
// receive.
func (c *Client) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	c.logger.Debugf("connection:Execute", "wsURL:%q method:%q", c.wsURL, method)
	id := atomic.AddInt64(&c.msgID, 1)

	// Setup event handler used to block for response to message being sent.
	recvCh := make(chan *cdproto.Message, 1)
	evCancelCtx, evCancelFn := context.WithCancel(ctx)
	msgCh := make(chan *cdproto.Message, 1)
	c.msgSubsMu.Lock()
	c.msgSubs[id] = msgCh
	c.msgSubsMu.Unlock()
	go func() {
		for {
			select {
			case <-evCancelCtx.Done():
				c.logger.Debugf("Connection:Execute:<-evCancelCtx.Done()", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
				return
			case msg := <-msgCh:
				select {
				case <-evCancelCtx.Done():
					c.logger.Debugf("Client:Execute:<-evCancelCtx.Done()#2", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
				case recvCh <- msg:
					// We expect only one response with the matching message ID,
					// then remove event handler by cancelling context and stopping goroutine.
					evCancelFn()
					return
				}
			}
		}
	}()
	// c.onAll(evCancelCtx, chEvHandler)
	defer evCancelFn() // Remove event handler

	// Send the message
	var buf []byte
	if params != nil {
		var err error
		buf, err = easyjson.Marshal(params)
		if err != nil {
			return err
		}
	}
	msg := &cdproto.Message{
		ID:     id,
		Method: cdproto.MethodType(method),
		Params: buf,
	}

	// We use different sessions to send messages to "targets"
	// (browser, page, frame etc.) in CDP.
	//
	// If we don't specify a session (a session ID in the JSON message),
	// it will be a message for the browser target.
	//
	// With a session ID set in the context (WithSessionID(ctx)),
	// it will properly route the CDP message to the correct target
	// (page, frame, etc.).
	if sid := GetSessionID(ctx); sid != "" {
		msg.SessionID = target.SessionID(sid)
	}

	return c.send(ctx, msg, recvCh, res)
}

// TODO: Wrap this in a friendlier method
func (c *Client) ExecuteWithoutExpectationOnReply(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	sid := GetSessionID(ctx)
	// c.logger.Debugf("Session:ExecuteWithoutExpectationOnReply", "sid:%v tid:%v method:%q", s.id, s.targetID, method)
	// Certain methods aren't available to the user directly.
	if method == target.CommandCloseTarget {
		return errors.New("to close the target, cancel its context")
	}
	// if s.crashed {
	// 	s.logger.Debugf("Session:ExecuteWithoutExpectationOnReply", "sid:%v tid:%v method:%q, ErrTargetCrashed", s.id, s.targetID, method)
	// 	return ErrTargetCrashed
	// }

	// c.logger.Debugf("Session:Execute:s.conn.send", "sid:%v tid:%v method:%q", s.id, s.targetID, method)

	var buf []byte
	if params != nil {
		var err error
		buf, err = easyjson.Marshal(params)
		if err != nil {
			// s.logger.Debugf("Session:ExecuteWithoutExpectationOnReply:Marshal", "sid:%v tid:%v method:%q err=%v", s.id, s.targetID, method, err)
			return err
		}
	}
	msg := &cdproto.Message{
		ID:        atomic.AddInt64(&c.msgID, 1),
		SessionID: target.SessionID(sid),
		Method:    cdproto.MethodType(method),
		Params:    buf,
	}

	return c.send(contextWithDoneChan(ctx, c.done), msg, nil, res)
}

// Subscribe returns a channel that will be notified when the provided CDP
// events are received for the given session and frame IDs, and a cancellation
// function that will unsubscribe and close the channel.
func (c *Client) Subscribe(
	ctx context.Context, frameID string, events ...cdproto.MethodType,
) (<-chan *Event, func()) {
	sessionID := GetSessionID(ctx)
	return c.watcher.subscribe(sessionID, frameID, events...)
}

func (c *Client) send(ctx context.Context, msg *cdproto.Message, recvCh chan *cdproto.Message, res easyjson.Unmarshaler) error {
	select {
	case c.sendCh <- msg:
	case err := <-c.errorCh:
		c.logger.Debugf("Connection:send:<-c.errorCh", "wsURL:%q sid:%v, err:%v", c.wsURL, msg.SessionID, err)
		var wsErr wsIOError
		if errors.As(err, &wsErr) {
			return c.conn.handleIOError(wsErr.Unwrap())
		}
		return err
	// 	return err
	// case code := <-c.closeCh:
	// 	c.logger.Debugf("Connection:send:<-c.closeCh", "wsURL:%q sid:%v, websocket code:%v", c.wsURL, msg.SessionID, code)
	// 	_ = c.conn.close(code)
	// 	return &websocket.CloseError{Code: code}
	// case <-c.done:
	// 	c.logger.Debugf("Connection:send:<-c.done", "wsURL:%q sid:%v", c.wsURL, msg.SessionID)
	// 	return nil
	case <-ctx.Done():
		c.logger.Errorf("Connection:send:<-ctx.Done()", "wsURL:%q sid:%v err:%v", c.wsURL, msg.SessionID, c.ctx.Err())
		return ctx.Err()
	case <-c.ctx.Done():
		c.logger.Errorf("Connection:send:<-c.ctx.Done()", "wsURL:%q sid:%v err:%v", c.wsURL, msg.SessionID, c.ctx.Err())
		return ctx.Err()
	}

	// Block waiting for response.
	if recvCh == nil {
		return nil
	}
	select {
	case msg := <-recvCh:
		switch {
		case msg == nil:
			c.logger.Debugf("Connection:send", "wsURL:%q, err:ErrChannelClosed", c.wsURL)
			return errors.New("msg is nil")
		case msg.Error != nil:
			// c.logger.Debugf("Connection:send", "sid:%v tid:%v wsURL:%q, msg err:%v", sid, tid, c.wsURL, msg.Error)
			return msg.Error
		case res != nil:
			return easyjson.Unmarshal(msg.Result, res)
		}
	case err := <-c.errorCh:
		// c.logger.Debugf("Connection:send:<-c.errorCh #2", "sid:%v tid:%v wsURL:%q, err:%v", msg.SessionID, tid, c.wsURL, err)
		var wsErr wsIOError
		if errors.As(err, &wsErr) {
			return c.conn.handleIOError(wsErr.Unwrap())
		}
		return err
	// case code := <-c.closeCh:
	// 	// c.logger.Debugf("Connection:send:<-c.closeCh #2", "sid:%v tid:%v wsURL:%q, websocket code:%v", msg.SessionID, tid, c.wsURL, code)
	// 	_ = c.conn.close(code)
	// 	return &websocket.CloseError{Code: code}
	// case <-c.done:
	// 	c.logger.Debugf("Connection:send:<-c.done #2", "sid:%v tid:%v wsURL:%q", msg.SessionID, tid, c.wsURL)
	case <-ctx.Done():
		// c.logger.Debugf("Connection:send:<-ctx.Done()", "sid:%v tid:%v wsURL:%q err:%v", msg.SessionID, tid, c.wsURL, c.ctx.Err())
		return ctx.Err()
	case <-c.ctx.Done():
		// c.logger.Debugf("Connection:send:<-c.ctx.Done()", "sid:%v tid:%v wsURL:%q err:%v", msg.SessionID, tid, c.wsURL, c.ctx.Err())
		return c.ctx.Err()
	}

	return nil
}

func (c *Client) recvLoop() {
	for {
		fmt.Printf(">>> looping in Client.recvLoop()\n")
		msg, err := c.conn.readMessage()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				c.logger.Errorf("Client:recvLoop", "wsURL:%q ioErr:%v", c.wsURL, err)
				c.conn.handleIOError(err)
			}
			fmt.Printf(">>> got err in Client.recvLoop(): %#+v\n", err)
			return
		}

		msgParams, _ := msg.Params.MarshalJSON()
		fmt.Printf(">>> got message in Client.recvLoop(): ID: %d, SessionID: %q, Method: %s, Params: %s\n", msg.ID, msg.SessionID, msg.Method, msgParams)
		switch {
		case msg.Method != "":
			evt, err := cdproto.UnmarshalMessage(msg)
			if err != nil {
				c.logger.Errorf("cdp", "unmarshalling CDP message: %w", err)
				continue
			}
			// Try to extract the frame ID if it exists
			var p struct {
				FrameID cdpext.FrameID `json:"frameId"`
			}
			_ = json.Unmarshal(msgParams, &p)
			c.watcher.notify(&Event{
				Name:      msg.Method,
				Data:      evt,
				sessionID: msg.SessionID,
				frameID:   p.FrameID,
			})
		case msg.ID > 0:
			fmt.Printf(">>> received message with ID %d\n", msg.ID)
			// TODO: Move this to the watcher?
			c.msgSubsMu.Lock()
			ch := c.recvCh
			if idCh, ok := c.msgSubs[msg.ID]; ok {
				ch = idCh
				delete(c.msgSubs, msg.ID)
			}
			c.msgSubsMu.Unlock()
			select {
			case ch <- msg:
			case <-c.ctx.Done():
				c.logger.Errorf("cdp", "receiving CDP messages from %q: %v", c.ctx.Err())
				return
			}
		default:
			c.logger.Errorf("cdp", "ignoring malformed incoming CDP message (missing id or method): %#v (message: %s)", msg, msg.Error.Message)
		}
	}
}

func (c *Client) recvMsgLoop() {
	for {
		select {
		case msg := <-c.recvCh:
			msgParams, _ := msg.Params.MarshalJSON()
			fmt.Printf(">>> got message in Client.recvMsgLoop(): ID: %d, SessionID: %q, Method: %q, Params: %s\n", msg.ID, msg.SessionID, msg.Method, msgParams)
		case <-c.ctx.Done():
			c.logger.Debugf("Client:recvMsgLoop", "returning, ctx.Err: %q", c.ctx.Err())
			return
		}
	}
}

func (c *Client) sendLoop() {
	// c.logger.Debugf("Client:sendLoop", "wsURL:%q, starts", c.wsURL)
	for {
		fmt.Printf(">>> looping in Client.sendLoop()\n")
		select {
		case msg := <-c.sendCh:
			fmt.Printf(">>> writing message with ID %d in Client.sendLoop()\n", msg.ID)
			err := c.conn.writeMessage(msg)
			if err != nil {
				fmt.Printf(">>> got err writing message with ID %d in Client.sendLoop()\n", msg.ID)
				c.errorCh <- err
			}
		// case code := <-c.closeCh:
		// 	c.logger.Debugf("Client:sendLoop:<-c.closeCh", "wsURL:%q code:%d", c.wsURL, code)
		// 	_ = c.conn.close(code)
		// 	return
		case <-c.done:
			c.logger.Debugf("Client:sendLoop:<-c.done#2", "wsURL:%q", c.wsURL)
			return
		case <-c.ctx.Done():
			c.logger.Debugf("Client:sendLoop", "returning, ctx.Err: %q", c.ctx.Err())
			c.conn.Close()
			return
		}
	}
}
