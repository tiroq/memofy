package obsws

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RecordingState represents OBS recording status
type RecordingState struct {
	Recording   bool      `json:"recording"`
	StartTime   time.Time `json:"start_time"`
	Duration    int       `json:"duration_seconds"` // Seconds since start
	OutputPath  string    `json:"output_path"`
	OBSStatus   string    `json:"obs_status"` // "connected", "disconnected", "error"
	OBSVersion  string    `json:"obs_version"`
	LastUpdated time.Time `json:"last_updated"`
}

// Client represents an OBS WebSocket v5 client
type Client struct {
	url        string
	password   string
	conn       *websocket.Conn
	mu         sync.RWMutex
	connected  bool
	identified bool
	requestID  int
	responses  map[int]chan *Response
	responseMu sync.RWMutex

	// Event handlers
	onRecordStateChanged func(recording bool)
	onDisconnected       func()

	// Recording state cache
	recordingState RecordingState
	stateMu        sync.RWMutex

	// Reconnection
	reconnectEnabled bool
	reconnectDelay   time.Duration
	stopChan         chan struct{}
}

// Message types
type Message struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}

type HelloData struct {
	OBSWebSocketVersion string `json:"obsWebSocketVersion"`
	RPCVersion          int    `json:"rpcVersion"`
	Authentication      struct {
		Challenge string `json:"challenge"`
		Salt      string `json:"salt"`
	} `json:"authentication"`
}

type IdentifyData struct {
	RPCVersion         int    `json:"rpcVersion"`
	Authentication     string `json:"authentication,omitempty"`
	EventSubscriptions int    `json:"eventSubscriptions"`
}

type Request struct {
	RequestType string      `json:"requestType"`
	RequestID   string      `json:"requestId"`
	RequestData interface{} `json:"requestData,omitempty"`
}

type Response struct {
	RequestType   string `json:"requestType"`
	RequestID     string `json:"requestId"`
	RequestStatus struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Comment string `json:"comment,omitempty"`
	} `json:"requestStatus"`
	ResponseData json.RawMessage `json:"responseData,omitempty"`
}

type Event struct {
	EventType string          `json:"eventType"`
	EventData json.RawMessage `json:"eventData,omitempty"`
}

// OpCodes for WebSocket protocol
const (
	OpHello                = 0
	OpIdentify             = 1
	OpIdentified           = 2
	OpReidentify           = 3
	OpEvent                = 5
	OpRequest              = 6
	OpRequestResponse      = 7
	OpRequestBatch         = 8
	OpRequestBatchResponse = 9
)

// Event subscription flags
const (
	EventSubscriptionAll = 0xFFFFFFFF
)

// NewClient creates a new OBS WebSocket client
func NewClient(url, password string) *Client {
	return &Client{
		url:              url,
		password:         password,
		responses:        make(map[int]chan *Response),
		reconnectEnabled: true,
		reconnectDelay:   5 * time.Second,
		stopChan:         make(chan struct{}),
		recordingState: RecordingState{
			OBSStatus:   "disconnected",
			LastUpdated: time.Now(),
		},
	}
}

// Connect establishes WebSocket connection and authenticates
func (c *Client) Connect() error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		c.updateOBSStatus("disconnected", "")
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	// Start message reader
	go c.readMessages()

	// Wait for Hello message (with timeout)
	helloChan := make(chan *HelloData, 1)
	errChan := make(chan error, 1)

	go func() {
		// Read first message (should be Hello)
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			errChan <- err
			return
		}

		if msg.Op != OpHello {
			errChan <- fmt.Errorf("expected Hello (op=0), got op=%d", msg.Op)
			return
		}

		var hello HelloData
		if err := json.Unmarshal(msg.D, &hello); err != nil {
			errChan <- err
			return
		}

		helloChan <- &hello
	}()

	select {
	case hello := <-helloChan:
		return c.authenticate(hello)
	case err := <-errChan:
		c.disconnect()
		return err
	case <-time.After(10 * time.Second):
		c.disconnect()
		return fmt.Errorf("timeout waiting for Hello message")
	}
}

// authenticate sends Identify message with auth response
func (c *Client) authenticate(hello *HelloData) error {
	identify := IdentifyData{
		RPCVersion:         1,
		EventSubscriptions: EventSubscriptionAll,
	}

	// If authentication required, generate auth string
	if hello.Authentication.Challenge != "" && c.password != "" {
		// secret = base64(sha256(password + salt))
		secret := sha256.Sum256([]byte(c.password + hello.Authentication.Salt))
		secretB64 := base64.StdEncoding.EncodeToString(secret[:])

		// auth = base64(sha256(secret + challenge))
		auth := sha256.Sum256([]byte(secretB64 + hello.Authentication.Challenge))
		identify.Authentication = base64.StdEncoding.EncodeToString(auth[:])
	}

	msg := Message{
		Op: OpIdentify,
	}
	msg.D, _ = json.Marshal(identify)

	c.mu.RLock()
	err := c.conn.WriteJSON(msg)
	c.mu.RUnlock()

	if err != nil {
		c.disconnect()
		return err
	}

	// Wait for Identified response
	identifiedChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		var msg Message
		c.mu.RLock()
		err := c.conn.ReadJSON(&msg)
		c.mu.RUnlock()

		if err != nil {
			errChan <- err
			return
		}

		if msg.Op == OpIdentified {
			identifiedChan <- true
		} else {
			errChan <- fmt.Errorf("expected Identified (op=2), got op=%d", msg.Op)
		}
	}()

	select {
	case <-identifiedChan:
		c.mu.Lock()
		c.identified = true
		c.mu.Unlock()
		c.updateOBSStatus("connected", hello.OBSWebSocketVersion)
		return nil
	case err := <-errChan:
		c.disconnect()
		return err
	case <-time.After(10 * time.Second):
		c.disconnect()
		return fmt.Errorf("timeout waiting for Identified message")
	}
}

// readMessages continuously reads and dispatches WebSocket messages
func (c *Client) readMessages() {
	defer func() {
		c.disconnect()
		if c.reconnectEnabled {
			c.reconnect()
		}
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		var msg Message
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		if err := conn.ReadJSON(&msg); err != nil {
			if c.onDisconnected != nil {
				c.onDisconnected()
			}
			return
		}

		switch msg.Op {
		case OpEvent:
			var event Event
			if err := json.Unmarshal(msg.D, &event); err == nil {
				c.handleEvent(&event)
			}

		case OpRequestResponse:
			var resp Response
			if err := json.Unmarshal(msg.D, &resp); err == nil {
				c.handleResponse(&resp)
			}
		}
	}
}

// handleEvent processes OBS events
func (c *Client) handleEvent(event *Event) {
	switch event.EventType {
	case "RecordStateChanged":
		var data struct {
			OutputActive bool   `json:"outputActive"`
			OutputPath   string `json:"outputPath"`
		}
		if err := json.Unmarshal(event.EventData, &data); err == nil {
			c.stateMu.Lock()
			c.recordingState.Recording = data.OutputActive
			c.recordingState.OutputPath = data.OutputPath
			if data.OutputActive {
				c.recordingState.StartTime = time.Now()
			}
			c.recordingState.LastUpdated = time.Now()
			c.stateMu.Unlock()

			if c.onRecordStateChanged != nil {
				c.onRecordStateChanged(data.OutputActive)
			}
		}
	}
}

// handleResponse routes responses to waiting request channels
func (c *Client) handleResponse(resp *Response) {
	c.responseMu.RLock()
	defer c.responseMu.RUnlock()

	// Parse request ID
	var id int
	if _, err := fmt.Sscanf(resp.RequestID, "%d", &id); err != nil {
		log.Printf("Warning: failed to parse request ID: %v", err)
		return
	}

	if ch, ok := c.responses[id]; ok {
		ch <- resp
	}
}

// sendRequest sends a request and waits for response
func (c *Client) sendRequest(requestType string, requestData interface{}) (*Response, error) {
	c.mu.RLock()
	if !c.connected || !c.identified {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	c.requestID++
	id := c.requestID
	requestID := fmt.Sprintf("%d", id)

	req := Request{
		RequestType: requestType,
		RequestID:   requestID,
		RequestData: requestData,
	}

	msg := Message{
		Op: OpRequest,
	}
	msg.D, _ = json.Marshal(req)

	// Create response channel
	respChan := make(chan *Response, 1)
	c.responseMu.Lock()
	c.responses[id] = respChan
	c.responseMu.Unlock()

	defer func() {
		c.responseMu.Lock()
		delete(c.responses, id)
		c.responseMu.Unlock()
	}()

	// Send request
	c.mu.RLock()
	err := c.conn.WriteJSON(msg)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if !resp.RequestStatus.Result {
			return nil, fmt.Errorf("request failed: %s (code %d)", resp.RequestStatus.Comment, resp.RequestStatus.Code)
		}
		return resp, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// disconnect closes the WebSocket connection
func (c *Client) disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("Warning: failed to close connection: %v", err)
		}
		c.conn = nil
	}
	c.connected = false
	c.identified = false

	c.updateOBSStatus("disconnected", "")
}

// reconnect attempts to reconnect with exponential backoff
func (c *Client) reconnect() {
	delay := c.reconnectDelay
	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(delay):
			if err := c.Connect(); err == nil {
				return // Successfully reconnected
			}
			// Exponential backoff (max 60s)
			delay *= 2
			if delay > 60*time.Second {
				delay = 60 * time.Second
			}
		}
	}
}

// updateOBSStatus updates the OBS connection status
func (c *Client) updateOBSStatus(status, version string) {
	c.stateMu.Lock()
	c.recordingState.OBSStatus = status
	c.recordingState.OBSVersion = version
	c.recordingState.LastUpdated = time.Now()
	c.stateMu.Unlock()
}

// Disconnect gracefully closes connection and stops reconnection
func (c *Client) Disconnect() {
	c.reconnectEnabled = false
	close(c.stopChan)
	c.disconnect()
}

// SetReconnectEnabled enables/disables automatic reconnection
func (c *Client) SetReconnectEnabled(enabled bool) {
	c.reconnectEnabled = enabled
}

// OnRecordStateChanged registers callback for recording state changes
func (c *Client) OnRecordStateChanged(handler func(recording bool)) {
	c.onRecordStateChanged = handler
}

// OnDisconnected registers callback for disconnection events
func (c *Client) OnDisconnected(handler func()) {
	c.onDisconnected = handler
}

// IsConnected returns current connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.identified
}
