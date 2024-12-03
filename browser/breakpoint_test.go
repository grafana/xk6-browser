package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type breakpointTest struct {
	updated      []breakpoint
	resumeCalled bool
}

func (bpt *breakpointTest) update(breakpoints []breakpoint) {
	bpt.updated = breakpoints
}

func (bpt *breakpointTest) resume() {
	bpt.resumeCalled = true
}

func newBreakpointClientTest(
	t *testing.T, serverHandler func(conn *websocket.Conn),
) (*breakpointClient, *breakpointTest) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var upgrader websocket.Upgrader
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() {
			if err := conn.Close(); err != nil {
				t.Logf("closing websocket connection: %v", err)
			}
		}()
		serverHandler(conn)
	}))

	var breakpoints breakpointTest
	client, err := dialBreakpointServer(context.Background(), "ws://"+srv.Listener.Addr().String(), &breakpoints)
	require.NoError(t, err)

	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		if err := client.close(); err != nil {
			t.Logf("closing client connection: %v", err)
		}
	})

	return client, &breakpoints
}

func TestBreakpointClient(t *testing.T) {
	handlerDone := make(chan struct{})
	client, breakpoints := newBreakpointClientTest(t, func(conn *websocket.Conn) {
		defer close(handlerDone)
		message := map[string]any{
			"command": "update_breakpoints",
			"data": []breakpoint{
				{File: "file1.js", Line: 10},
				{File: "file2.js", Line: 20},
			},
		}
		messageBytes, _ := json.Marshal(message)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, messageBytes))

		// Simulate sending a resume message
		message = map[string]any{
			"command": "resume",
		}
		messageBytes, _ = json.Marshal(message)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, messageBytes))
	})
	go client.listen()

	select {
	case <-handlerDone:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for server to handle the pause message")
	}
	require.Len(t, breakpoints.updated, 2)
	assert.Equal(t, "file1.js", breakpoints.updated[0].File)
	assert.Equal(t, 10, breakpoints.updated[0].Line)
	assert.Equal(t, "file2.js", breakpoints.updated[1].File)
	assert.Equal(t, 20, breakpoints.updated[1].Line)
	assert.True(t, breakpoints.resumeCalled)
}

func TestBreakpointClient_SendPause(t *testing.T) {
	handlerDone := make(chan struct{})
	client, _ := newBreakpointClientTest(t, func(conn *websocket.Conn) {
		defer close(handlerDone)

		_, message, err := conn.ReadMessage()
		require.NoError(t, err)

		var envelope map[string]any
		err = json.Unmarshal(message, &envelope)
		require.NoError(t, err)

		assert.Equal(t, "pause", envelope["type"])
	})

	require.NoError(t, client.sendPause())
	select {
	case <-handlerDone:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for server to handle the pause message")
	}
}
