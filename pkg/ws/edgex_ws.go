package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EdgeXWSClient handles WebSocket connection to EdgeX
type EdgeXWSClient struct {
	url         string
	conn        *websocket.Conn
	mu          sync.RWMutex
	handlers    map[string]func(json.RawMessage)
	stopCh      chan struct{}
	reconnectCh chan struct{}
}

type EdgeXWSMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Time    string          `json:"time,omitempty"`
}

type EdgeXTickerContent struct {
	DataType string        `json:"dataType"`
	Channel  string        `json:"channel"`
	Data     []EdgeXTicker `json:"data"`
}

type EdgeXTicker struct {
	ContractId string `json:"contractId"`
	LastPrice  string `json:"lastPrice"`
	IndexPrice string `json:"indexPrice"`
	MarkPrice  string `json:"markPrice"`
	// Add more fields as needed
}

func NewEdgeXWSClient(url string) *EdgeXWSClient {
	return &EdgeXWSClient{
		url:         url,
		handlers:    make(map[string]func(json.RawMessage)),
		stopCh:      make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
	}
}

func (c *EdgeXWSClient) Connect(ctx context.Context) error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to EdgeX WebSocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	log.Printf("EdgeX WebSocket connected to %s", c.url)

	// Start message handler
	go c.handleMessages()

	// Start ping/pong handler
	go c.handlePingPong()

	return nil
}

func (c *EdgeXWSClient) Subscribe(channel string, handler func(json.RawMessage)) error {
	c.mu.Lock()
	c.handlers[channel] = handler
	c.mu.Unlock()

	msg := EdgeXWSMessage{
		Type:    "subscribe",
		Channel: channel,
	}

	return c.sendMessage(msg)
}

func (c *EdgeXWSClient) Unsubscribe(channel string) error {
	c.mu.Lock()
	delete(c.handlers, channel)
	c.mu.Unlock()

	msg := EdgeXWSMessage{
		Type:    "unsubscribe",
		Channel: channel,
	}

	return c.sendMessage(msg)
}

func (c *EdgeXWSClient) sendMessage(msg interface{}) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	return conn.WriteJSON(msg)
}

func (c *EdgeXWSClient) handleMessages() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				time.Sleep(time.Second)
				continue
			}

			var msg EdgeXWSMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("EdgeX WS read error: %v", err)
				// Trigger reconnect
				select {
				case c.reconnectCh <- struct{}{}:
				default:
				}
				time.Sleep(time.Second)
				continue
			}

			// Handle different message types
			switch msg.Type {
			case "ping":
				// Respond with pong
				pong := EdgeXWSMessage{
					Type: "pong",
					Time: msg.Time,
				}
				c.sendMessage(pong)

			case "pong":
				// Ignore pong responses

			case "subscribed":
				log.Printf("EdgeX WS subscribed to channel: %s", msg.Channel)

			case "quote-event":
				// Handle quote events
				c.mu.RLock()
				handler, ok := c.handlers[msg.Channel]
				c.mu.RUnlock()

				if ok && handler != nil {
					handler(msg.Content)
				}

			case "error":
				log.Printf("EdgeX WS error: %s", string(msg.Content))
			}
		}
	}
}

func (c *EdgeXWSClient) handlePingPong() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// Send ping
			ping := EdgeXWSMessage{
				Type: "ping",
				Time: fmt.Sprintf("%d", time.Now().UnixMilli()),
			}
			if err := c.sendMessage(ping); err != nil {
				log.Printf("EdgeX WS ping error: %v", err)
			}
		}
	}
}

func (c *EdgeXWSClient) Close() error {
	close(c.stopCh)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}
