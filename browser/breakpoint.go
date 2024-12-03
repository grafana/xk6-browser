package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/grafana/xk6-browser/env"
)

/*
Protocol:

- get_breakpoints: Client initially requests the breakpoints from the server.
	- Example: [{"file":"file:///Users/inanc/grafana/k6browser/main/examples/fillform.js", "line": 28}]
- update_breakpoints: Server sends the updated breakpoints to the client.
	- Example: [{"file":"file:///Users/inanc/grafana/k6browser/main/examples/fillform.js", "line": 32}]
- resume: Server sends a message to the client to resume the script execution.
	- Example: {"command":"resume"}

Client:

- The client pauses the script execution when a breakpoint is hit.
- The server should send the "resume" message to the client to resume the script execution.
- The client continuously listens for messages from the server.

Example Run:

- K6_BROWSER_BREAKPOINT_SERVER_URL=ws://localhost:8080/breakpoint k6 run script.js
*/

type breakpoint struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Condition string `json:"condition,omitempty"`
}

type breakpointRegistry struct {
	muBreakpoints sync.RWMutex
	breakpoints   []breakpoint
	pauser        chan chan struct{}
	client        *breakpointClient
}

func newBreakpointRegistry() *breakpointRegistry {
	return &breakpointRegistry{
		// breakpoints: []breakpoint{
		// 	{
		// 		File: "file:///Users/inanc/grafana/k6browser/main/examples/fillform.js",
		// 		Line: 26,
		// 	},
		// 	{
		// 		File: "file:///Users/inanc/grafana/k6browser/main/examples/fillform.js",
		// 		Line: 32,
		// 	},
		// },
		pauser: make(chan chan struct{}, 1),
	}
}

func (br *breakpointRegistry) update(breakpoints []breakpoint) {
	br.muBreakpoints.Lock()
	defer br.muBreakpoints.Unlock()

	br.breakpoints = breakpoints
}

func (br *breakpointRegistry) matches(p position) (breakpoint, bool) {
	br.muBreakpoints.RLock()
	defer br.muBreakpoints.RUnlock()

	// We need to compare between /path/to/test-script.js and file:///path/to/test-script.js
	for _, b := range br.breakpoints {
		if strings.Contains(p.Filename, b.File) && b.Line == p.Line {
			return b, true
		}
	}

	return breakpoint{}, false
}

// pause pauses the script execution.
func (br *breakpointRegistry) pause(b breakpoint, column int, funcName string) error {
	if err := br.client.sendPause(b, column, funcName); err != nil {
		return err
	}

	c := make(chan struct{})
	br.pauser <- c
	<-c

	return nil
}

// resume resumes the script execution.
func (br *breakpointRegistry) resume() {
	c := <-br.pauser
	close(c)
}

// pauseOnBreakpoint is a helper that pauses the script execution
// when a breakpoint is hit in the script.
func pauseOnBreakpoint(vu moduleVU) {
	bp := vu.breakpointRegistry
	if bp == nil { // breakpoints are disabled
		return
	}

	pos := getCurrentLineNumber(vu)
	log.Printf("current line: %v", pos)

	b, ok := bp.matches(pos)
	if !ok {
		return
	}

	log.Printf("pausing at %v:%v", pos.Filename, pos.Line)
	if err := bp.pause(b, pos.Column, pos.FuncName); err != nil {
		log.Printf("failed to pause: %v", err)
	}
}

type breakpointUpdateResumer interface {
	update(breakpoints []breakpoint)
	resume()
}

type breakpointClient struct {
	conn     *websocket.Conn
	registry breakpointUpdateResumer
}

func dialBreakpointServer(
	ctx context.Context, serverURL string, registry breakpointUpdateResumer, retryCount int,
) (*breakpointClient, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("breakpointClient: parsing websocket server URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil && strings.Contains(err.Error(), "connection refused") && retryCount > 0 {
		time.Sleep(time.Millisecond * 500)
		return dialBreakpointServer(ctx, serverURL, registry, retryCount-1)
	}
	if err != nil {
		return nil, fmt.Errorf("breakpointClient: dialing server: %w", err)
	}

	client := &breakpointClient{
		conn:     conn,
		registry: registry,
	}

	return client, nil
}

func (bc *breakpointClient) listen() {
	for {
		_, message, err := bc.conn.ReadMessage()
		if websocket.IsCloseError(err,
			websocket.CloseAbnormalClosure,
			websocket.CloseNormalClosure,
			websocket.CloseGoingAway,
		) {
			return
		}
		if err != nil {
			log.Printf("breakpointClient: reading websocket message: %v", err)
			return
		}
		log.Println("breakpointClient: received websocket message:", string(message))

		var envelope struct {
			Command string          `json:"command"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message, &envelope); err != nil {
			log.Printf("breakpointClient: unmarshaling breakpoint message: %v", err)
			continue
		}

		switch envelope.Command {
		case "update_breakpoints":
			bc.updateBreakpoints(envelope.Data)
		case "resume":
			bc.handleResume()
		default:
			log.Printf("breakpointClient: unknown command: %s", envelope.Command)
		}
	}
}

func (bc *breakpointClient) updateBreakpoints(data []byte) {
	var breakpoints []breakpoint
	if err := json.Unmarshal(data, &breakpoints); err != nil {
		log.Printf("breakpointClient: parsing breakpoints: %v", err)
		return
	}
	bc.registry.update(breakpoints)
}

func (bc *breakpointClient) handleResume() {
	bc.registry.resume()
}

func (bc *breakpointClient) sendPause(b breakpoint, column int, funcName string) error {
	envelope := map[string]interface{}{
		"event": "pause",
		"location": map[string]interface{}{
			"file":     b.File,
			"line":     b.Line,
			"column":   column,
			"funcName": funcName,
		},
	}

	message, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("breakpointClient: marshaling pause message: %w", err)
	}
	if err := bc.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("breakpointClient: sending pause message: %w", err)
	}

	return nil
}

func (bc *breakpointClient) close() error {
	if err := bc.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	); err != nil {
		return fmt.Errorf("breakpointClient: sending websocket close message: %w", err)
	}
	if err := bc.conn.Close(); err != nil {
		return fmt.Errorf("breakpointClient: closing websocket connection: %w", err)
	}
	return nil
}

func parseBreakpointServerURL(envLookup env.LookupFunc) string {
	v, _ := envLookup(env.BreakpointServerURL)
	return strings.TrimSpace(v)
}
