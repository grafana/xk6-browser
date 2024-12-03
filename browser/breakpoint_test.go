package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type breakpointTest struct {
	mu           sync.Mutex
	updated      []breakpoint
	resumeCalled bool
}

func (bpt *breakpointTest) update(breakpoints []breakpoint) {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	bpt.updated = breakpoints
}

func (bpt *breakpointTest) resume() {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	bpt.resumeCalled = true
}

func (bpt *breakpointTest) all() []breakpoint {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	return bpt.updated
}

func (bpt *breakpointTest) isResumeCalled() bool {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	return bpt.resumeCalled
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
	time.Sleep(5 * time.Second) // TODO: find a better way to wait for the message to be processed
	items := breakpoints.all()
	require.Len(t, items, 2)
	assert.Equal(t, "file1.js", items[0].File)
	assert.Equal(t, 10, items[0].Line)
	assert.Equal(t, "file2.js", items[1].File)
	assert.Equal(t, 20, items[1].Line)
	assert.True(t, breakpoints.isResumeCalled())
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

		assert.Equal(t, "pause", envelope["command"])
	})

	require.NoError(t, client.sendPause())
	select {
	case <-handlerDone:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for server to handle the pause message")
	}
}
