package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type breakpointUpdateResumer interface {
	update(breakpoints []breakpoint)
	resume()
}

type breakpointClient struct {
	conn     *websocket.Conn
	registry breakpointUpdateResumer
}

func dialBreakpointServer(
	ctx context.Context, serverURL string, registry breakpointUpdateResumer,
) (*breakpointClient, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("breakpointClient: parsing websocket server URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
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
			bc.handleUpdateBreakpoints(envelope.Data)
		case "resume":
			bc.handleResume()
		default:
			log.Printf("breakpointClient: unknown command: %s", envelope.Command)
		}
	}
}

func (bc *breakpointClient) handleUpdateBreakpoints(data []byte) {
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

func (bc *breakpointClient) sendPause() error {
	envelope := map[string]any{
		"type": "pause",
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
