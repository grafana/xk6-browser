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
	"github.com/grafana/xk6-browser/log"
	"github.com/mailru/easyjson"

	cdppage "github.com/chromedp/cdproto/page"
)

var _ cdp.Executor = &Client{}

// Client manages CDP communication with the browser.
type Client struct {
	ctx    context.Context
	logger *log.Logger

	sessionsMu sync.RWMutex
	sessions   map[target.SessionID]*session

	conn  *connection
	msgID int64
	wsURL string
}

// NewClient returns a new Client.
func NewClient(ctx context.Context, logger *log.Logger) *Client {
	return &Client{ctx: ctx, logger: logger}
}

// Connect to the browser that exposes a CDP API at wsURL.
func (c *Client) Connect(wsURL string) (err error) {
	if c.conn, err = newConnection(c.ctx, wsURL, c.logger); err != nil {
		return
	}
	c.logger.Infof("cdp", "established CDP connection to %q", wsURL)

	go c.recvLoop(c.ctx)

	return nil
}

// Execute implements cdproto.Executor and performs a synchronous send and receive.
func (c *Client) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	c.logger.Debugf("connection:Execute", "wsURL:%q method:%q", c.wsURL, method)
	id := atomic.AddInt64(&c.msgID, 1)

	// Setup event handler used to block for response to message being sent.
	ch := make(chan *cdproto.Message, 1)
	evCancelCtx, evCancelFn := context.WithCancel(ctx)
	chEvHandler := make(chan Event)
	go func() {
		for {
			select {
			case <-evCancelCtx.Done():
				c.logger.Debugf("connection:Execute:<-evCancelCtx.Done()", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
				return
			case ev := <-chEvHandler:
				msg, ok := ev.data.(*cdproto.Message)
				if ok && msg.ID == id {
					select {
					case <-evCancelCtx.Done():
						c.logger.Debugf("connection:Execute:<-evCancelCtx.Done()#2", "wsURL:%q err:%v", c.wsURL, evCancelCtx.Err())
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
	c.onAll(evCancelCtx, chEvHandler)
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
	return c.send(c.ctx, msg, ch, res)
}

func (c *Client) Navigate(url, frameID, referrer string) (string, error) {
	action := cdppage.Navigate(url).WithReferrer(referrer).WithFrameID(cdp.FrameID(frameID))
	_, documentID, errorText, err := action.Do(cdp.WithExecutor(c.ctx, c))
	if err != nil {
		err = fmt.Errorf("%s at %q: %w", errorText, url, err)
	}

	return documentID.String(), err
}

// Send a CDP command to the browser without waiting for a response.
// func (c *Client) send(action action) error {
// 	return nil
// }

func (c *Client) recvLoop(ctx context.Context) {
	for {
		msg, err := c.conn.readMessage()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.logger.Infof("cdp", "connection closed to %q: %w", c.wsURL, err)
				return
			}
			c.logger.Errorf("cdp", "reading CDP message: %w", err)
		}

		ev, err := cdproto.UnmarshalMessage(&msg)
		if err != nil {
			c.logger.Errorf("cdp", "unmarshalling CDP message: %w", err)
			continue
		}

		// TODO: Move this to an EventWatcher
		// Handle attachment and detachment from targets,
		// creating and deleting sessions as necessary.
		if msg.Method == cdproto.EventTargetAttachedToTarget {
			eva := ev.(*target.EventAttachedToTarget)
			sid, tid := eva.SessionID, eva.TargetInfo.TargetID

			c.sessionsMu.Lock()
			session := NewSession(c.ctx, c, sid, tid, c.logger)
			c.logger.Debugf("Connection:recvLoop:EventAttachedToTarget", "sid:%v tid:%v wsURL:%q", sid, tid, c.wsURL)
			c.sessions[sid] = session
			c.sessionsMu.Unlock()
		} else if msg.Method == cdproto.EventTargetDetachedFromTarget {
			ev, err := cdproto.UnmarshalMessage(&msg)
			if err != nil {
				c.logger.Errorf("cdp", "%s", err)
				continue
			}
			evt := ev.(*target.EventDetachedFromTarget)
			sid := evt.SessionID
			tid := c.findTargetIDForLog(sid)
			c.closeSession(sid, tid)
		}

		switch {
		case msg.SessionID != "" && (msg.Method != "" || msg.ID != 0):
			// TODO: possible data race - session can get removed after getting it here
			session := c.getSession(msg.SessionID)
			if session == nil {
				continue
			}
			if msg.Error != nil && msg.Error.Message == "No session with given id" {
				c.logger.Debugf("Connection:recvLoop", "sid:%v tid:%v wsURL:%q, closeSession #2", session.id, session.targetID, c.wsURL)
				c.closeSession(session.id, session.targetID)
				continue
			}

			select {
			case session.readCh <- &msg:
			case code := <-c.closeCh:
				c.logger.Debugf("Connection:recvLoop:<-c.closeCh", "sid:%v tid:%v wsURL:%v crashed:%t", session.id, session.targetID, c.wsURL, session.crashed)
				_ = c.close(code)
			case <-c.done:
				c.logger.Debugf("Connection:recvLoop:<-c.done", "sid:%v tid:%v wsURL:%v crashed:%t", session.id, session.targetID, c.wsURL, session.crashed)
				return
			}

		case msg.Method != "":
			c.logger.Debugf("Connection:recvLoop:msg.Method:emit", "sid:%v method:%q", msg.SessionID, msg.Method)
			ev, err := cdproto.UnmarshalMessage(&msg)
			if err != nil {
				c.logger.Errorf("cdp", "%s", err)
				continue
			}
			c.emit(string(msg.Method), ev)

		case msg.ID != 0:
			c.logger.Debugf("Connection:recvLoop:msg.ID:emit", "sid:%v method:%q", msg.SessionID, msg.Method)
			c.emit("", &msg)

		default:
			c.logger.Errorf("cdp", "ignoring malformed incoming message (missing id or method): %#v (message: %s)", msg, msg.Error.Message)
		}
	}
}
