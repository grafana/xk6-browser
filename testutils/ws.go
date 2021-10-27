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

// Package testutils is indended only for use in tests, do not import in production code!
package testutils

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	k6lib "go.k6.io/k6/lib"
	k6netext "go.k6.io/k6/lib/netext"
	k6types "go.k6.io/k6/lib/types"
)

const httpDomain = "wsbin.local"

// WSTestServer can be used as a test alternative to a real CDP compatible browser.
type WSTestServer struct {
	Mux           *http.ServeMux
	ServerHTTP    *httptest.Server
	Dialer        *k6netext.Dialer
	HTTPTransport *http.Transport
	Context       context.Context
	Cleanup       func()
}

func getWebsocketHandlerAbnormalClosure() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, req, w.Header())
		if err != nil {
			return
		}
		err = conn.Close() // This forces a connection closure without a proper WS close message exchange
		if err != nil {
			return
		}
	})
}

func getWebsocketHandlerEcho() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, req, w.Header())
		if err != nil {
			return
		}
		messageType, r, e := conn.NextReader()
		if e != nil {
			return
		}
		var wc io.WriteCloser
		wc, err = conn.NextWriter(messageType)
		if err != nil {
			return
		}
		if _, err = io.Copy(wc, r); err != nil {
			return
		}
		if err = wc.Close(); err != nil {
			return
		}
		err = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(10*time.Second),
		)
		if err != nil {
			return
		}
	})
}

// NewWSTestServerWithCustomHandler creates a WS test server with abnormal closure behavior
func NewWSTestServerWithClosureAbnormal(t testing.TB) *WSTestServer {
	return NewWSTestServer(t, "/closure-abnormal", getWebsocketHandlerAbnormalClosure())
}

// NewWSTestServerWithCustomHandler creates a WS test server with an echo handler
func NewWSTestServerWithEcho(t testing.TB) *WSTestServer {
	return NewWSTestServer(t, "/echo", getWebsocketHandlerEcho())
}

// NewWSTestServer returns a fully configured and running WS test server
func NewWSTestServer(t testing.TB, path string, handler http.Handler) *WSTestServer {
	// Create a http.ServeMux and set the httpbin handler as the default
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle("/", httpbin.New().Handler())

	// Initialize the HTTP server and get its details
	httpSrv := httptest.NewServer(mux)
	httpURL, err := url.Parse(httpSrv.URL)
	require.NoError(t, err)
	httpIP := net.ParseIP(httpURL.Hostname())
	require.NotNil(t, httpIP)

	httpDomainValue, err := k6lib.NewHostAddress(httpIP, "")
	require.NoError(t, err)

	// Set up the dialer with shorter timeouts and the custom domains
	dialer := k6netext.NewDialer(net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 10 * time.Second,
		DualStack: true,
	}, k6netext.NewResolver(net.LookupIP, 0, k6types.DNSfirst, k6types.DNSpreferIPv4))
	dialer.Hosts = map[string]*k6lib.HostAddress{
		httpDomain: httpDomainValue,
	}

	// Pre-configure the HTTP client transport with the dialer and TLS config (incl. HTTP2 support)
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	require.NoError(t, http2.ConfigureTransport(transport))

	ctx, ctxCancel := context.WithCancel(context.Background())
	return &WSTestServer{
		Mux:           mux,
		ServerHTTP:    httpSrv,
		Dialer:        dialer,
		HTTPTransport: transport,
		Context:       ctx,
		Cleanup: func() {
			httpSrv.Close()
			ctxCancel()
		},
	}
}
