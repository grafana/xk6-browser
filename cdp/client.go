package cdp

import (
	"context"
	"errors"
	"net"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/target"
	"github.com/grafana/xk6-browser/log"
)

// Client manages CDP communication with the browser.
type Client struct {
	logger *log.Logger
	conn   *connection
}

// NewClient returns a new Client.
func NewClient(logger *log.Logger) *Client {
	return &Client{logger: logger}
}

// Connect to the browser that exposes a CDP API at wsURL.
func (c *Client) Connect(ctx context.Context, wsURL string) (err error) {
	if c.conn, err = newConnection(ctx, wsURL, c.logger); err != nil {
		return
	}
	c.logger.Infof("cdp", "established CDP connection to %q", wsURL)

	return nil
}

// Send a CDP command to the browser without waiting for a response.
func (c *Client) Send(cmd *Command) error {
}

func (c *Client) recvLoop() {
	for {
		msg, err := c.conn.readMessage()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
			}
		}

		// TODO: Move this to an EventWatcher
		// Handle attachment and detachment from targets,
		// creating and deleting sessions as necessary.
		if msg.Method == cdproto.EventTargetAttachedToTarget {
			ev, err := cdproto.UnmarshalMessage(msg)
			if err != nil {
				c.logger.Errorf("cdp", "%s", err)
				continue
			}
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
