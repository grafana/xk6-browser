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

	"github.com/grafana/xk6-browser/log"
	"github.com/grafana/xk6-browser/tests/ws"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/gorilla/websocket"
	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionCreateSession(t *testing.T) {
	const (
		cdpTargetID         = "target_id_0123456789"
		cdpBrowserContextID = "browser_context_id_0123456789"

		targetAttachedToTargetEvent = `
		{
			"sessionId": "session_id_0123456789",
			"targetInfo": {
				"targetId": "target_id_0123456789",
				"type": "page",
				"title": "",
				"url": "about:blank",
				"attached": true,
				"browserContextId": "browser_context_id_0123456789"
			},
			"waitingForDebugger": false
		}`

		targetAttachedToTargetResult = `
		{
			"sessionId":"session_id_0123456789"
		}
		`
	)

	cmdsReceived := make([]cdproto.MethodType, 0)
	handler := func(conn *websocket.Conn, msg *cdproto.Message, writeCh chan cdproto.Message, done chan struct{}) {
		if msg.SessionID != "" && msg.Method != "" {
			switch msg.Method {
			case cdproto.MethodType(cdproto.CommandPageEnable):
				writeCh <- cdproto.Message{
					ID:        msg.ID,
					SessionID: msg.SessionID,
				}
				close(done) // We're done after receiving the Page.enable command
			}
		} else if msg.Method != "" {
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
					Params: easyjson.RawMessage([]byte(targetAttachedToTargetEvent)),
				}
				writeCh <- cdproto.Message{
					ID:        msg.ID,
					SessionID: msg.SessionID,
					Result:    easyjson.RawMessage([]byte(targetAttachedToTargetResult)),
				}
			}
		}
	}

	server := ws.NewServer(t, ws.WithCDPHandler("/cdp", handler, &cmdsReceived))

	t.Run("send and recv session commands", func(t *testing.T) {
		ctx := context.Background()
		url, _ := url.Parse(server.ServerHTTP.URL)
		wsURL := fmt.Sprintf("ws://%s/cdp", url.Host)
		conn, err := NewConnection(ctx, wsURL, log.NewNullLogger())

		if assert.NoError(t, err) {
			session, err := conn.createSession(&target.Info{
				Type:             "page",
				TargetID:         cdpTargetID,
				BrowserContextID: cdpBrowserContextID,
			})

			if assert.NoError(t, err) {
				action := cdppage.Enable()
				err := action.Do(cdp.WithExecutor(ctx, session))

				require.NoError(t, err)
				require.Equal(t, []cdproto.MethodType{
					cdproto.CommandTargetAttachToTarget,
					cdproto.CommandPageEnable,
				}, cmdsReceived)
			}

			conn.Close()
		}
	})
}
