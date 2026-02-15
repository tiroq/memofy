package testutil

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MockOBSServer simulates an OBS WebSocket v5 server for testing
type MockOBSServer struct {
	listener    net.Listener
	server      *http.Server
	conn        *websocket.Conn
	responses   map[string]interface{}
	mode        string
	mu          sync.Mutex
	connected   bool
	fixturesDir string
}

// FailureModes define how the mock server behaves
const (
	ModeNormal     = "normal"
	ModeCode204    = "code204"
	ModeCode203    = "code203"
	ModeTimeout    = "timeout"
	ModeDisconnect = "disconnect"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewMockOBS creates a new mock OBS server
func NewMockOBS() *MockOBSServer {
	return &MockOBSServer{
		responses:   make(map[string]interface{}),
		mode:        ModeNormal,
		fixturesDir: "../internal/obsws/testdata",
	}
}

// Start begins listening on a dynamic port
func (m *MockOBSServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	m.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleWebSocket)

	m.server = &http.Server{Handler: mux}

	go func() {
		_ = m.server.Serve(m.listener)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	return nil
}

// Stop gracefully shuts down the server
func (m *MockOBSServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
	}

	if m.server != nil {
		_ = m.server.Close()
	}

	if m.listener != nil {
		_ = m.listener.Close()
	}

	m.connected = false
	return nil
}

// Addr returns the server's listening address
func (m *MockOBSServer) Addr() string {
	if m.listener == nil {
		return ""
	}
	return m.listener.Addr().String()
}

// SetFailureMode configures how the server responds to requests
func (m *MockOBSServer) SetFailureMode(mode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

// QueueResponse queues a specific response for a request type
func (m *MockOBSServer) QueueResponse(requestType string, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[requestType] = response
}

// LoadFixture loads a JSON fixture file
func (m *MockOBSServer) LoadFixture(filename string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/%s", m.fixturesDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture %s: %w", filename, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse fixture %s: %w", filename, err)
	}

	return result, nil
}

// handleWebSocket manages the WebSocket connection
func (m *MockOBSServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	m.mu.Lock()
	m.conn = conn
	m.connected = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.connected = false
		m.mu.Unlock()
		_ = conn.Close()
	}()

	// Send Hello message (op 0)
	hello, err := m.LoadFixture("hello_response.json")
	if err != nil {
		return
	}
	if err := conn.WriteJSON(hello); err != nil {
		return
	}

	// Wait for Identify message (op 1)
	var identifyMsg map[string]interface{}
	if err := conn.ReadJSON(&identifyMsg); err != nil {
		return
	}

	// Send Identified message (op 2)
	identified, err := m.LoadFixture("identified_response.json")
	if err != nil {
		return
	}
	if err := conn.WriteJSON(identified); err != nil {
		return
	}

	// Handle subsequent requests
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		response := m.generateResponse(msg)
		if response == nil {
			continue
		}

		// Simulate delay if in timeout mode
		m.mu.Lock()
		mode := m.mode
		m.mu.Unlock()

		if mode == ModeTimeout {
			time.Sleep(7 * time.Second) // Longer than typical 6s timeout
		}

		if mode == ModeDisconnect {
			// Just close the connection
			break
		}

		if err := conn.WriteJSON(response); err != nil {
			break
		}
	}
}

// generateResponse creates a response based on the request and current mode
func (m *MockOBSServer) generateResponse(msg map[string]interface{}) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract request details
	d, ok := msg["d"].(map[string]interface{})
	if !ok {
		return nil
	}

	requestType, _ := d["requestType"].(string)
	requestID, _ := d["requestId"].(string)

	// Check for queued response
	if queuedResp, exists := m.responses[requestType]; exists {
		if respMap, ok := queuedResp.(map[string]interface{}); ok {
			// Set the request ID
			if d, ok := respMap["d"].(map[string]interface{}); ok {
				d["requestId"] = requestID
			}
			return respMap
		}
	}

	// Generate response based on mode
	switch m.mode {
	case ModeCode204:
		fixture, _ := m.LoadFixture("code_204_response.json")
		if d, ok := fixture["d"].(map[string]interface{}); ok {
			d["requestId"] = requestID
			d["requestType"] = requestType
		}
		return fixture

	case ModeCode203:
		fixture, _ := m.LoadFixture("code_203_response.json")
		if d, ok := fixture["d"].(map[string]interface{}); ok {
			d["requestId"] = requestID
			d["requestType"] = requestType
		}
		return fixture

	default:
		// Normal mode - return success
		return map[string]interface{}{
			"op": 7,
			"d": map[string]interface{}{
				"requestType": requestType,
				"requestId":   requestID,
				"requestStatus": map[string]interface{}{
					"result":  true,
					"code":    100,
					"comment": "",
				},
				"responseData": map[string]interface{}{},
			},
		}
	}
}

// Connected returns whether a client is currently connected
func (m *MockOBSServer) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}
