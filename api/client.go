package api

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/url"
	"time"

	"wemon-agent/collector"
	"wemon-agent/config"

	"github.com/gorilla/websocket"
)

type Client struct {
	cfg       *config.Config
	collector *collector.Collector
	conn      *websocket.Conn
	interval  time.Duration
	done      chan struct{}
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type ConfigUpdate struct {
	IntervalSeconds int `json:"interval_seconds"`
}

// NewClient initializes the WebSocket client for communicating with the WeMon server.
func NewClient(cfg *config.Config, col *collector.Collector) *Client {
	return &Client{
		cfg:       cfg,
		collector: col,
		interval:  time.Duration(cfg.IntervalSeconds) * time.Second,
		done:      make(chan struct{}),
	}
}

// Start initiates the WebSocket connection loop with exponential backoff reconnect logic.
func (c *Client) Start() {
	u, err := url.Parse(c.cfg.ServerURL)
	if err != nil {
		log.Fatalf("Invalid server URL: %v", err)
	}

	scheme := "wss"
	if u.Scheme == "http" {
		scheme = "ws"
	}
	u.Scheme = scheme
	u.Path = "/api/agent/ws"

	// Add token to query params
	q := u.Query()
	q.Set("token", c.cfg.NodeToken)
	u.RawQuery = q.Encode()

	wsURL := u.String()

	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		log.Printf("Connecting to WeMon server at %s...", c.cfg.ServerURL)

		dialer := websocket.DefaultDialer
		if c.cfg.InsecureSkipVerify {
			dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("Connection failed: %v. Retrying in %v...", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		log.Println("Connected to WeMon server!")
		c.conn = conn
		backoff = 5 * time.Second // Reset backoff on success

		connDone := make(chan struct{})

		// Start reader for server control messages
		go c.readPump(connDone)

		// Start writer for sending metrics
		c.writePump(connDone)

		c.conn.Close()
		log.Println("Disconnected from server. Attempting to reconnect...")
	}
}

func (c *Client) readPump(connDone chan struct{}) {
	defer close(connDone)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		switch msg.Type {
		case "config_update":
			var update ConfigUpdate
			if err := json.Unmarshal(msg.Data, &update); err == nil && update.IntervalSeconds > 0 {
				log.Printf("Received config update: interval changed to %d seconds", update.IntervalSeconds)
				c.interval = time.Duration(update.IntervalSeconds) * time.Second
			}
		default:
			log.Printf("Unknown message type received: %s", msg.Type)
		}
	}
}

func (c *Client) writePump(connDone chan struct{}) {
	currentInterval := c.interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		// Recreate ticker if interval was updated dynamically by the server
		if currentInterval != c.interval {
			currentInterval = c.interval
			ticker.Stop()
			ticker = time.NewTicker(currentInterval)
			log.Printf("Metric collection ticker updated to %v", currentInterval)
		}

		select {
		case <-connDone:
			return
		case <-ticker.C:
			metrics, err := c.collector.Collect()
			if err != nil {
				log.Printf("Error collecting metrics: %v", err)
				continue
			}

			metricsJSON, err := json.Marshal(metrics)
			if err != nil {
				log.Printf("Error marshaling metrics: %v", err)
				continue
			}

			msg := Message{
				Type: "metrics",
				Data: metricsJSON,
			}

			payload, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshaling payload: %v", err)
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}
