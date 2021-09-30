/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package common

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/gorilla/websocket"
	"github.com/grafana/xk6-browser/testutils"
	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnection(t *testing.T) {
	server := testutils.NewWSTestServerWithEcho(t)
	defer server.Cleanup()

	t.Run("connect", func(t *testing.T) {
		ctx := context.Background()
		url, _ := url.Parse(server.ServerHTTP.URL)
		wsURL := fmt.Sprintf("ws://%s/echo", url.Host)
		conn, err := NewConnection(ctx, wsURL, NewLogger(ctx, NullLogger(), false, nil))
		conn.Close()

		require.NoError(t, err)
	})
}

func TestConnectionClosureAbnormal(t *testing.T) {
	server := testutils.NewWSTestServerWithClosureAbnormal(t)
	defer server.Cleanup()

	t.Run("closure abnormal", func(t *testing.T) {
		ctx := context.Background()
		url, _ := url.Parse(server.ServerHTTP.URL)
		wsURL := fmt.Sprintf("ws://%s/closure-abnormal", url.Host)
		conn, err := NewConnection(ctx, wsURL, NewLogger(ctx, NullLogger(), false, nil))

		if assert.NoError(t, err) {
			action := target.SetDiscoverTargets(true)
			err := action.Do(cdp.WithExecutor(ctx, conn))
			require.EqualError(t, err, "websocket: close 1006 (abnormal closure): unexpected EOF")
		}
	})
}

func TestConnectionSendRecv(t *testing.T) {
	server := testutils.NewWSTestServerWithCDPHandler(t, testutils.CDPDefaultHandler, nil)
	defer server.Cleanup()

	t.Run("send command with empty reply", func(t *testing.T) {
		ctx := context.Background()
		url, _ := url.Parse(server.ServerHTTP.URL)
		wsURL := fmt.Sprintf("ws://%s/cdp", url.Host)
		conn, err := NewConnection(ctx, wsURL, NewLogger(ctx, NullLogger(), false, nil))

		if assert.NoError(t, err) {
			action := target.SetDiscoverTargets(true)
			err := action.Do(cdp.WithExecutor(ctx, conn))
			require.NoError(t, err)
		}
	})
}

func TestConnectionCreateSession(t *testing.T) {
	cmdsReceived := make([]cdproto.MethodType, 0)
	handler := func(conn *websocket.Conn, msg *cdproto.Message, writeCh chan cdproto.Message, done chan struct{}) {
		if msg.SessionID == "" && msg.Method != "" {
			switch msg.Method {
			case cdproto.MethodType(cdproto.CommandTargetSetDiscoverTargets):
				writeCh <- cdproto.Message{
					ID:        msg.ID,
					SessionID: msg.SessionID,
					Result:    easyjson.RawMessage([]byte("{}")),
				}
			case cdproto.MethodType(cdproto.CommandTargetAttachToTarget):
				switch msg.Method {
				case cdproto.MethodType(cdproto.CommandTargetSetDiscoverTargets):
					writeCh <- cdproto.Message{
						ID:        msg.ID,
						SessionID: msg.SessionID,
						Result:    easyjson.RawMessage([]byte("{}")),
					}
				case cdproto.MethodType(cdproto.CommandTargetAttachToTarget):
					writeCh <- cdproto.Message{
						Method: cdproto.EventTargetAttachedToTarget,
						Params: easyjson.RawMessage([]byte(`
                            {
                                "sessionId": "0123456789",
                                "targetInfo": {
                                    "targetId": "abcdef0123456789",
                                    "type": "page",
                                    "title": "",
                                    "url": "about:blank",
                                    "attached": true,
                                    "browserContextId": "0123456789876543210"
                                },
                                "waitingForDebugger": false
                            }
                        `)),
					}
					writeCh <- cdproto.Message{
						ID:        msg.ID,
						SessionID: msg.SessionID,
						Result:    easyjson.RawMessage([]byte(`{"sessionId":"0123456789"}`)),
					}
				}
			}
		}
	}

	server := testutils.NewWSTestServerWithCDPHandler(t, handler, &cmdsReceived)
	defer server.Cleanup()

	t.Run("create session for target", func(t *testing.T) {
		ctx := context.Background()
		url, _ := url.Parse(server.ServerHTTP.URL)
		wsURL := fmt.Sprintf("ws://%s/cdp", url.Host)
		conn, err := NewConnection(ctx, wsURL, NewLogger(ctx, NullLogger(), false, nil))

		if assert.NoError(t, err) {
			session, err := conn.createSession(&target.Info{
				TargetID:         "abcdef0123456789",
				Type:             "page",
				BrowserContextID: "0123456789876543210",
			})

			require.NoError(t, err)
			require.NotNil(t, session)
			require.NotEmpty(t, session.id)
			require.NotEmpty(t, conn.sessions)
			require.Len(t, conn.sessions, 1)
			require.Equal(t, conn.sessions[session.id], session)
			require.Equal(t, []cdproto.MethodType{
				cdproto.CommandTargetAttachToTarget,
			}, cmdsReceived)
		}
	})
}
