package cdp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/grafana/xk6-browser/cdp/event"
	"github.com/grafana/xk6-browser/log"
	"github.com/mailru/easyjson"

	cdppage "github.com/chromedp/cdproto/page"
)

var _ cdp.Executor = &Client{}

// Client manages CDP communication with the browser.
type Client struct {
	ctx    context.Context
	logger *log.Logger

	conn   *connection
	msgID  int64
	recvCh chan *cdproto.Message
	sendCh chan *cdproto.Message
	// closeCh chan int
	errorCh chan error
	done    chan struct{}

	sessionsMu sync.RWMutex
	sessions   map[target.SessionID]*session
	watcher    *event.Watcher
	wsURL      string
}

// NewClient returns a new Client that is unusable until a CDP connection is
// established with Connect().
func NewClient(ctx context.Context, logger *log.Logger) *Client {
	return &Client{
		ctx:    ctx,
		logger: logger,
		// Buffered channels to avoid blocking in Execute
		recvCh:  make(chan *cdproto.Message, 32),
		sendCh:  make(chan *cdproto.Message, 32),
		msgID:   1000,
		watcher: event.NewWatcher(ctx),
	}
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
	go c.sendLoop()

	return nil
}

// Execute implements cdproto.Executor and performs a synchronous send and
// receive.
func (c *Client) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	c.logger.Debugf("connection:Execute", "wsURL:%q method:%q", c.wsURL, method)
	id := atomic.AddInt64(&c.msgID, 1)

	// Setup event handler used to block for response to message being sent.
	ch := make(chan *cdproto.Message, 1)
	evCancelCtx, evCancelFn := context.WithCancel(ctx)
	// chEvHandler := make(chan Event)
	go func() {
		for {
			select {
			case <-evCancelCtx.Done():
				c.logger.Debugf("Connection:Execute:<-evCancelCtx.Done()", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
				return
			case msg := <-c.recvCh:
				if msg.ID == id {
					select {
					case <-evCancelCtx.Done():
						c.logger.Debugf("Client:Execute:<-evCancelCtx.Done()#2", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
					case ch <- msg:
						// We expect only one response with the matching message ID,
						// then remove event handler by cancelling context and stopping goroutine.
						evCancelFn()
						return
					}
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

	if sid := getSessionID(ctx); sid != "" {
		msg.SessionID = target.SessionID(sid)
	}

	return c.send(ctx, msg, ch, res)
}

// Navigate sends the Page.navigate CDP command.
// TODO: Break this up into CDP domains.
func (c *Client) Navigate(url, frameID, referrer string) (string, error) {
	action := cdppage.Navigate(url).WithReferrer(referrer).WithFrameID(cdp.FrameID(frameID))
	_, documentID, errorText, err := action.Do(cdp.WithExecutor(c.ctx, c))
	if err != nil {
		err = fmt.Errorf("%s at %q: %w", errorText, url, err)
	}

	return documentID.String(), err
}

// Subscribe returns a channel that will be notified when the provided CDP event
// is received.
func (c *Client) Subscribe(evt event.CDPName) <-chan *event.Event {
	return c.watcher.Subscribe(evt)
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
		// var sid target.SessionID
		// tid = ""
		// if msg != nil {
		// 	sid = msg.SessionID
		// tid = c.findTargetIDForLog(sid)
		// }
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

		fmt.Printf(">>> got message in Client.recvLoop(): %#+v\n", msg)
		switch {
		case msg.Method != "":
			evt, err := cdproto.UnmarshalMessage(msg)
			if err != nil {
				c.logger.Errorf("cdp", "unmarshalling CDP message: %w", err)
				continue
			}
			fmt.Printf(">>> received event %s\n", msg.Method)
			c.watcher.OnEventReceived(&event.Event{event.CDPName(msg.Method), evt})
		case msg.ID != 0:
			fmt.Printf(">>> received message with ID %d\n", msg.ID)
			select {
			// TODO: Add a timeout?
			case c.recvCh <- msg:
			case <-c.ctx.Done():
				c.logger.Errorf("cdp", "receiving CDP messages from %q: %v", c.ctx.Err())
				return
			}
		default:
			c.logger.Errorf("cdp", "ignoring malformed incoming CDP message (missing id or method): %#v (message: %s)", msg, msg.Error.Message)
		}

		// TODO: Move this to an EventWatcher
		// Handle attachment and detachment from targets,
		// creating and deleting sessions as necessary.
		// if msg.Method == cdproto.EventTargetAttachedToTarget {
		// 	eva := ev.(*target.EventAttachedToTarget)
		// 	sid, tid := eva.SessionID, eva.TargetInfo.TargetID

		// 	c.sessionsMu.Lock()
		// 	session := NewSession(c.ctx, c, sid, tid, c.logger)
		// 	c.logger.Debugf("Connection:recvLoop:EventAttachedToTarget", "sid:%v tid:%v wsURL:%q", sid, tid, c.wsURL)
		// 	c.sessions[sid] = session
		// 	c.sessionsMu.Unlock()
		// } else if msg.Method == cdproto.EventTargetDetachedFromTarget {
		// 	ev, err := cdproto.UnmarshalMessage(&msg)
		// 	if err != nil {
		// 		c.logger.Errorf("cdp", "%s", err)
		// 		continue
		// 	}
		// 	evt := ev.(*target.EventDetachedFromTarget)
		// 	sid := evt.SessionID
		// 	tid := c.findTargetIDForLog(sid)
		// 	c.closeSession(sid, tid)
		// }

		// switch {
		// case msg.SessionID != "" && (msg.Method != "" || msg.ID != 0):
		// 	// TODO: possible data race - session can get removed after getting it here
		// 	session := c.getSession(msg.SessionID)
		// 	if session == nil {
		// 		continue
		// 	}
		// 	if msg.Error != nil && msg.Error.Message == "No session with given id" {
		// 		c.logger.Debugf("Connection:recvLoop", "sid:%v tid:%v wsURL:%q, closeSession #2", session.id, session.targetID, c.wsURL)
		// 		c.closeSession(session.id, session.targetID)
		// 		continue
		// 	}

		// 	select {
		// 	case session.readCh <- &msg:
		// 	case code := <-c.closeCh:
		// 		c.logger.Debugf("Connection:recvLoop:<-c.closeCh", "sid:%v tid:%v wsURL:%v crashed:%t", session.id, session.targetID, c.wsURL, session.crashed)
		// 		_ = c.close(code)
		// 	case <-c.done:
		// 		c.logger.Debugf("Connection:recvLoop:<-c.done", "sid:%v tid:%v wsURL:%v crashed:%t", session.id, session.targetID, c.wsURL, session.crashed)
		// 		return
		// 	}

		// case msg.Method != "":
		// 	c.logger.Debugf("Connection:recvLoop:msg.Method:emit", "sid:%v method:%q", msg.SessionID, msg.Method)
		// 	ev, err := cdproto.UnmarshalMessage(&msg)
		// 	if err != nil {
		// 		c.logger.Errorf("cdp", "%s", err)
		// 		continue
		// 	}
		// 	c.emit(string(msg.Method), ev)

		// // case msg.ID != 0:
		// // 	c.logger.Debugf("Connection:recvLoop:msg.ID:emit", "sid:%v method:%q", msg.SessionID, msg.Method)
		// // 	c.emit("", &msg)

		// default:
		// 	c.logger.Errorf("cdp", "ignoring malformed incoming message (missing id or method): %#v (message: %s)", msg, msg.Error.Message)
		// }
	}
}

func (c *Client) sendLoop() {
	// c.logger.Debugf("Client:sendLoop", "wsURL:%q, starts", c.wsURL)
	for {
		fmt.Printf(">>> looping in Client.sendLoop()\n")
		select {
		case msg := <-c.sendCh:
			fmt.Printf(">>> writing message in Client.sendLoop()\n")
			err := c.conn.writeMessage(msg)
			if err != nil {
				fmt.Printf(">>> got err writing message in Client.sendLoop()\n")
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
