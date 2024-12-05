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

func (bpt *breakpointTest) resume(stepOut bool) {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	bpt.resumeCalled = true
	_ = stepOut
}

func (bpt *breakpointTest) vars() []map[string]debugVarFunc {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	return nil // TODO interface pollution
}

func (bpt *breakpointTest) setStepOverMode(on bool) {
	bpt.mu.Lock()
	defer bpt.mu.Unlock()
	_ = on
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
	client, err := dialBreakpointServer(context.Background(), "ws://"+srv.Listener.Addr().String(), &breakpoints, 1)
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
	t.SkipNow()

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

	require.NoError(t, client.sendPause(breakpoint{}, 0, "page.goto"))
	select {
	case <-handlerDone:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for server to handle the pause message")
	}
}

func TestBreakpoint_Stepover(t *testing.T) {
	t.Run("stepover", func(t *testing.T) {
		reg := newBreakpointRegistry()
		reg.setStepOverMode(true)
		bp, ok := reg.matches(position{Filename: "foo.js", Line: 1})
		assert.True(t, ok)
		assert.Equal(t, breakpoint{File: "foo.js", Line: 1}, bp)
	})

	t.Run("stepover_off", func(t *testing.T) {
		reg := newBreakpointRegistry()
		reg.setStepOverMode(false)
		_, ok := reg.matches(position{Filename: "foo.js", Line: 1})
		assert.False(t, ok)
	})

	t.Run("stepover_off_with_breakpoint", func(t *testing.T) {
		reg := newBreakpointRegistry()
		reg.setStepOverMode(false)
		reg.update([]breakpoint{{File: "foo.js", Line: 1}})
		bp, ok := reg.matches(position{Filename: "foo.js", Line: 1})
		assert.True(t, ok)
		assert.Equal(t, breakpoint{File: "foo.js", Line: 1}, bp)
	})

	t.Run("pause_stepover_on", func(t *testing.T) {
		reg := newBreakpointRegistry()
		reg.setStepOverMode(true)

		bp := breakpoint{File: "foo.js", Line: 1}

		err := make(chan error)
		go func() {
			err <- reg.pause(bp, 0, "bar")
		}()
		go func() {
			reg.resume(false)
		}()
		select {
		case err := <-err:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for pause to be called")
		}
		require.True(t, reg.stepOverMode)
	})
	t.Run("pause_stepover_off", func(t *testing.T) {
		reg := newBreakpointRegistry()
		reg.setStepOverMode(true)

		bp := breakpoint{File: "foo.js", Line: 1}

		err := make(chan error)
		go func() {
			err <- reg.pause(bp, 0, "bar")
		}()
		go func() {
			reg.resume(true)
		}()
		select {
		case err := <-err:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for pause to be called")
		}
		require.False(t, reg.stepOverMode)
	})
}
