package cdp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/xk6-browser/log"

	"github.com/chromedp/cdproto"
	"github.com/gorilla/websocket"
	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
)

const wsWriteBufferSize = 1 << 20

type wsIOError struct{ error }

func (e wsIOError) Unwrap() error {
	return e.error
}

// Ensure Connection implements the EventEmitter and Executor interfaces.
// var _ cdp.Executor = &Connection{}

// type executorEmitter interface {
// 	cdp.Executor
// 	EventEmitter
// }

// type connection interface {
// 	Close(...goja.Value)
// 	// getSession(target.SessionID) *Session
// }

// type session interface {
// 	ExecuteWithoutExpectationOnReply(context.Context, string, easyjson.Marshaler, easyjson.Unmarshaler) error
// 	ID() target.SessionID
// 	TargetID() target.ID
// 	Done() <-chan struct{}
// }

// Action is the general interface of an CDP action.
// type Action interface {
// 	Do(context.Context) error
// }

// ActionFunc is an adapter to allow regular functions to be used as an Action.
// type ActionFunc func(context.Context) error

// Do executes the func f using the provided context.
// func (f ActionFunc) Do(ctx context.Context) error {
// 	return f(ctx)
// }

// TODO: Update this.
/*
		Connection represents a WebSocket connection and the root "Browser Session".

		                                      ┌───────────────────────────────────────────────────────────────────┐
	                                          │                                                                   │
	                                          │                          Browser Process                          │
	                                          │                                                                   │
	                                          └───────────────────────────────────────────────────────────────────┘

┌───────────────────────────┐                                           │      ▲
│Reads JSON-RPC CDP messages│                                           │      │
│from WS connection and puts│                                           ▼      │
│ them on incoming queue of │             ┌───────────────────────────────────────────────────────────────────┐
│    target session, as     ├─────────────■                                                                   │
│   identified by message   │             │                       WebSocket Connection                        │
│   session ID. Messages    │             │                                                                   │
│ without a session ID are  │             └───────────────────────────────────────────────────────────────────┘
│considered to belong to the│                    │      ▲                                       │      ▲
│  root "Browser Session".  │                    │      │                                       │      │
└───────────────────────────┘                    ▼      │                                       ▼      │
┌───────────────────────────┐             ┌────────────────────┐                         ┌────────────────────┐
│  Handles CDP messages on  ├─────────────■                    │                         │                    │
│incoming queue and puts CDP│             │      Session       │      *  *  *  *  *      │      Session       │
│   messages on outgoing    │             │                    │                         │                    │
│ channel of WS connection. │             └────────────────────┘                         └────────────────────┘
└───────────────────────────┘                    │      ▲                                       │      ▲

	│      │                                       │      │
	▼      │                                       ▼      │

┌───────────────────────────┐             ┌────────────────────┐                         ┌────────────────────┐
│Registers with session as a├─────────────■                    │                         │                    │
│handler for a specific CDP │             │   Event Listener   │      *  *  *  *  *      │   Event Listener   │
│       Domain event.       │             │                    │                         │                    │
└───────────────────────────┘             └────────────────────┘                         └────────────────────┘.
*/

// connection handles the low-level WebSocket communication with the browser.
type connection struct {
	wsURL        string
	wsConn       *websocket.Conn
	shutdownOnce sync.Once

	// Reuse the easyjson structs to avoid allocs per Read/Write.
	decoder jlexer.Lexer
	encoder jwriter.Writer

	logger *log.Logger
}

// NewConnection creates a new browser.
func newConnection(ctx context.Context, wsURL string, logger *log.Logger) (*connection, error) {
	var header http.Header
	var tlsConfig *tls.Config
	wsd := websocket.Dialer{
		HandshakeTimeout: time.Second * 60,
		Proxy:            http.ProxyFromEnvironment, // TODO(fix): use proxy settings from launch options
		TLSClientConfig:  tlsConfig,
		WriteBufferSize:  wsWriteBufferSize,
	}

	conn, _, connErr := wsd.DialContext(ctx, wsURL, header)
	if connErr != nil {
		return nil, connErr
	}

	c := connection{
		wsURL:  wsURL,
		logger: logger,
		wsConn: conn,
	}

	return &c, nil
}

// close cleanly closes the WebSocket connection.
// Returns an error if sending the close control frame fails.
func (c *connection) close(code int) error {
	c.logger.Debugf("Connection:close", "code:%d", code)

	var err error
	c.shutdownOnce.Do(func() {
		defer func() {
			_ = c.wsConn.Close()

			// Stop the main control loop
			// close(c.done)
		}()

		err = c.wsConn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(code, ""),
			time.Now().Add(10*time.Second),
		)

		// c.sessionsMu.Lock()
		// for _, s := range c.sessions {
		// 	s.close()
		// 	delete(c.sessions, s.id)
		// }
		// c.sessionsMu.Unlock()
	})

	return err
}

// TODO: Client should handle Sessions.
// func (c *Connection) closeSession(sid target.SessionID, tid target.ID) {
// 	c.logger.Debugf("Connection:closeSession", "sid:%v tid:%v wsURL:%v", sid, tid, c.wsURL)
// 	c.sessionsMu.Lock()
// 	if session, ok := c.sessions[sid]; ok {
// 		session.close()
// 	}
// 	delete(c.sessions, sid)
// 	c.sessionsMu.Unlock()
// }

// func (c *Connection) createSession(info *target.Info) (*Session, error) {
// 	c.logger.Debugf("Connection:createSession", "tid:%v bctxid:%v type:%s", info.TargetID, info.BrowserContextID, info.Type)

// 	var sessionID target.SessionID
// 	var err error
// 	action := target.AttachToTarget(info.TargetID).WithFlatten(true)
// 	if sessionID, err = action.Do(cdp.WithExecutor(c.ctx, c)); err != nil {
// 		c.logger.Debugf("Connection:createSession", "tid:%v bctxid:%v type:%s err:%v", info.TargetID, info.BrowserContextID, info.Type, err)
// 		return nil, err
// 	}
// 	sess := c.getSession(sessionID)
// 	if sess == nil {
// 		c.logger.Warnf("Connection:createSession", "tid:%v bctxid:%v type:%s sid:%v, session is nil", info.TargetID, info.BrowserContextID, info.Type, sessionID)
// 	}
// 	return sess, nil
// }

func (c *connection) readMessage() (*cdproto.Message, error) {
	fmt.Printf(">>> calling wsConn.ReadMessage()\n")
	_, buf, err := c.wsConn.ReadMessage()
	if err != nil {
		fmt.Printf(">>> got err from wsConn.ReadMessage(): %#+v\n", err)
		return nil, err
	}
	fmt.Printf(">>> got message from wsConn.ReadMessage()\n")

	var msg cdproto.Message
	c.decoder = jlexer.Lexer{Data: buf}
	msg.UnmarshalEasyJSON(&c.decoder)
	if err := c.decoder.Error(); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (c *connection) writeMessage(msg *cdproto.Message) error {
	c.encoder = jwriter.Writer{}
	msg.MarshalEasyJSON(&c.encoder)
	if err := c.encoder.Error; err != nil {
		return err
		// sid := msg.SessionID
		// tid := c.findTargetIDForLog(sid)
		// select {
		// case c.errorCh <- err:
		// c.logger.Debugf("Connection:sendLoop:c.errorCh <- err", "sid:%v tid:%v wsURL:%q err:%v", sid, tid, c.wsURL, err)
		// case <-c.done:
		// 	// c.logger.Debugf("Connection:sendLoop:<-c.done", "sid:%v tid:%v wsURL:%q", sid, tid, c.wsURL)
		// 	return nil
		// }
	}

	buf, _ := c.encoder.BuildBytes()
	c.logger.Tracef("cdp:send", "-> %s", buf)
	writer, err := c.wsConn.NextWriter(websocket.TextMessage)
	if err != nil {
		return wsIOError{err}
	}
	if _, err := writer.Write(buf); err != nil {
		return wsIOError{err}
	}
	if err := writer.Close(); err != nil {
		return wsIOError{err}
	}

	return nil
}

func (c *connection) handleIOError(err error) error {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return err
	}

	code := websocket.CloseGoingAway
	if e, ok := err.(*websocket.CloseError); ok {
		code = e.Code
	}

	return c.close(code)
}

// func (c *Connection) getSession(id target.SessionID) *Session {
// 	c.sessionsMu.RLock()
// 	defer c.sessionsMu.RUnlock()

// 	return c.sessions[id]
// }

// findTragetIDForLog should only be used for logging purposes.
// It will return an empty string if logger.DebugMode is false.
// func (c *Connection) findTargetIDForLog(id target.SessionID) target.ID {
// 	if !c.logger.DebugMode() {
// 		return ""
// 	}
// 	s := c.getSession(id)
// 	if s == nil {
// 		return ""
// 	}
// 	return s.targetID
// }

// func (c *connection) send(ctx context.Context, msg *cdproto.Message, recvCh chan *cdproto.Message, res easyjson.Unmarshaler) error {
// 	select {
// 	case c.sendCh <- msg:
// 	case err := <-c.errorCh:
// 		c.logger.Debugf("Connection:send:<-c.errorCh", "wsURL:%q sid:%v, err:%v", c.wsURL, msg.SessionID, err)
// 		return err
// 	case code := <-c.closeCh:
// 		c.logger.Debugf("Connection:send:<-c.closeCh", "wsURL:%q sid:%v, websocket code:%v", c.wsURL, msg.SessionID, code)
// 		_ = c.close(code)
// 		return &websocket.CloseError{Code: code}
// 	case <-c.done:
// 		c.logger.Debugf("Connection:send:<-c.done", "wsURL:%q sid:%v", c.wsURL, msg.SessionID)
// 		return nil
// 	case <-ctx.Done():
// 		c.logger.Errorf("Connection:send:<-ctx.Done()", "wsURL:%q sid:%v err:%v", c.wsURL, msg.SessionID, c.ctx.Err())
// 		return ctx.Err()
// 	case <-c.ctx.Done():
// 		c.logger.Errorf("Connection:send:<-c.ctx.Done()", "wsURL:%q sid:%v err:%v", c.wsURL, msg.SessionID, c.ctx.Err())
// 		return ctx.Err()
// 	}

// 	// Block waiting for response.
// 	if recvCh == nil {
// 		return nil
// 	}
// 	tid := c.findTargetIDForLog(msg.SessionID)
// 	select {
// 	case msg := <-recvCh:
// 		var sid target.SessionID
// 		tid = ""
// 		if msg != nil {
// 			sid = msg.SessionID
// 			tid = c.findTargetIDForLog(sid)
// 		}
// 		switch {
// 		case msg == nil:
// 			c.logger.Debugf("Connection:send", "wsURL:%q, err:ErrChannelClosed", c.wsURL)
// 			return ErrChannelClosed
// 		case msg.Error != nil:
// 			c.logger.Debugf("Connection:send", "sid:%v tid:%v wsURL:%q, msg err:%v", sid, tid, c.wsURL, msg.Error)
// 			return msg.Error
// 		case res != nil:
// 			return easyjson.Unmarshal(msg.Result, res)
// 		}
// 	case err := <-c.errorCh:
// 		c.logger.Debugf("Connection:send:<-c.errorCh #2", "sid:%v tid:%v wsURL:%q, err:%v", msg.SessionID, tid, c.wsURL, err)
// 		return err
// 	case code := <-c.closeCh:
// 		c.logger.Debugf("Connection:send:<-c.closeCh #2", "sid:%v tid:%v wsURL:%q, websocket code:%v", msg.SessionID, tid, c.wsURL, code)
// 		_ = c.close(code)
// 		return &websocket.CloseError{Code: code}
// 	case <-c.done:
// 		c.logger.Debugf("Connection:send:<-c.done #2", "sid:%v tid:%v wsURL:%q", msg.SessionID, tid, c.wsURL)
// 	case <-ctx.Done():
// 		c.logger.Debugf("Connection:send:<-ctx.Done()", "sid:%v tid:%v wsURL:%q err:%v", msg.SessionID, tid, c.wsURL, c.ctx.Err())
// 		return ctx.Err()
// 	case <-c.ctx.Done():
// 		c.logger.Debugf("Connection:send:<-c.ctx.Done()", "sid:%v tid:%v wsURL:%q err:%v", msg.SessionID, tid, c.wsURL, c.ctx.Err())
// 		return c.ctx.Err()
// 	}
// 	return nil
// }

// func (c *connection) sendLoop() {
// 	c.logger.Debugf("Connection:sendLoop", "wsURL:%q, starts", c.wsURL)
// 	for {
// 		select {
// 		case msg := <-c.sendCh:
// 			c.encoder = jwriter.Writer{}
// 			msg.MarshalEasyJSON(&c.encoder)
// 			if err := c.encoder.Error; err != nil {
// 				sid := msg.SessionID
// 				tid := c.findTargetIDForLog(sid)
// 				select {
// 				case c.errorCh <- err:
// 					c.logger.Debugf("Connection:sendLoop:c.errorCh <- err", "sid:%v tid:%v wsURL:%q err:%v", sid, tid, c.wsURL, err)
// 				case <-c.done:
// 					c.logger.Debugf("Connection:sendLoop:<-c.done", "sid:%v tid:%v wsURL:%q", sid, tid, c.wsURL)
// 					return
// 				}
// 			}

// 			buf, _ := c.encoder.BuildBytes()
// 			c.logger.Tracef("cdp:send", "-> %s", buf)
// 			writer, err := c.wsConn.NextWriter(websocket.TextMessage)
// 			if err != nil {
// 				c.handleIOError(err)
// 				return
// 			}
// 			if _, err := writer.Write(buf); err != nil {
// 				c.handleIOError(err)
// 				return
// 			}
// 			if err := writer.Close(); err != nil {
// 				c.handleIOError(err)
// 				return
// 			}
// 		case code := <-c.closeCh:
// 			c.logger.Debugf("Connection:sendLoop:<-c.closeCh", "wsURL:%q code:%d", c.wsURL, code)
// 			_ = c.close(code)
// 			return
// 		case <-c.done:
// 			c.logger.Debugf("Connection:sendLoop:<-c.done#2", "wsURL:%q", c.wsURL)
// 			return
// 		case <-c.ctx.Done():
// 			c.logger.Debugf("connection:sendLoop", "returning, ctx.Err: %q", c.ctx.Err())
// 			return
// 		}
// 	}
// }

func (c *connection) Close() {
	code := websocket.CloseGoingAway
	// if len(args) > 0 {
	// 	code = int(args[0].ToInteger())
	// }
	c.logger.Debugf("connection:Close", "wsURL:%q code:%d", c.wsURL, code)
	_ = c.close(code)
}

// Execute implements cdproto.Executor and performs a synchronous send and receive.
// func (c *Connection) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
// 	c.logger.Debugf("connection:Execute", "wsURL:%q method:%q", c.wsURL, method)
// 	id := atomic.AddInt64(&c.msgID, 1)

// 	// Setup event handler used to block for response to message being sent.
// 	ch := make(chan *cdproto.Message, 1)
// 	evCancelCtx, evCancelFn := context.WithCancel(ctx)
// 	chEvHandler := make(chan Event)
// 	go func() {
// 		for {
// 			select {
// 			case <-evCancelCtx.Done():
// 				c.logger.Debugf("connection:Execute:<-evCancelCtx.Done()", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
// 				return
// 			case ev := <-chEvHandler:
// 				msg, ok := ev.data.(*cdproto.Message)
// 				if ok && msg.ID == id {
// 					select {
// 					case <-evCancelCtx.Done():
// 						c.logger.Debugf("connection:Execute:<-evCancelCtx.Done()#2", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
// 					case ch <- msg:
// 						// We expect only one response with the matching message ID,
// 						// then remove event handler by cancelling context and stopping goroutine.
// 						evCancelFn()
// 						return
// 					}
// 				}
// 			}
// 		}
// 	}()
// 	c.onAll(evCancelCtx, chEvHandler)
// 	defer evCancelFn() // Remove event handler

// 	// Send the message
// 	var buf []byte
// 	if params != nil {
// 		var err error
// 		buf, err = easyjson.Marshal(params)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	msg := &cdproto.Message{
// 		ID:     id,
// 		Method: cdproto.MethodType(method),
// 		Params: buf,
// 	}
// 	return c.send(c.ctx, msg, ch, res)
// }
