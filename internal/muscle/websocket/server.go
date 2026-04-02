package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"gamification/config"
	"github.com/gorilla/websocket"
)

// Upgrader configures WebSocket upgrade parameters
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		return true
	},
}

// Message types
const (
	MsgTypeConnect       = "connect"
	MsgTypeAck           = "ack"
	MsgTypeBadgeEarned   = "badge_earned"
	MsgTypePointsUpdated = "points_updated"
	MsgTypeUserStats     = "user_stats_updated"
	MsgTypeStreakUpdated = "streak_updated"
	MsgTypePing          = "ping"
	MsgTypePong          = "pong"
)

// Server represents the WebSocket server
type Server struct {
	config *config.Config

	// Connection management
	mu          sync.RWMutex
	connections map[string]*Client // userID -> client

	// HTTP server
	httpServer *http.Server

	// Channel for broadcasting events
	eventChan chan *BroadcastEvent

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Client represents a connected WebSocket client
type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	server *Server
	closed bool
	mu     sync.Mutex
}

// BroadcastEvent represents an event to broadcast to users
type BroadcastEvent struct {
	UserIDs []string
	Type    string
	Payload interface{}
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	UserID  string          `json:"user_id,omitempty"`
	Token   string          `json:"token,omitempty"`
	Status  string          `json:"status,omitempty"`
	Points  int             `json:"points,omitempty"`
	Streak  int             `json:"streak,omitempty"`
	Stats   json.RawMessage `json:"stats,omitempty"`
}

// BadgeEarnedPayload represents badge earned event payload
type BadgeEarnedPayload struct {
	BadgeID          string `json:"badge_id"`
	BadgeName        string `json:"badge_name"`
	Description      string `json:"description"`
	Points           int    `json:"points"`
	BadgeIcon        string `json:"badge_icon,omitempty"`
	EarnedAt         string `json:"earned_at,omitempty"`
	PointsAwarded    int    `json:"points_awarded,omitempty"`
	BadgeDescription string `json:"badge_description,omitempty"`
}

// NewServer creates a new WebSocket server
func NewServer(cfg *config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:      cfg,
		connections: make(map[string]*Client),
		eventChan:   make(chan *BroadcastEvent, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	log.Println("Starting WebSocket server...")

	// Create HTTP server with mux
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.httpServer = &http.Server{
		Addr:         s.config.WebSocketAddr(),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start broadcast handler
	go s.handleBroadcast()

	// Start HTTP server in goroutine
	go func() {
		log.Printf("WebSocket server listening on %s", s.config.WebSocketAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the WebSocket server
func (s *Server) Stop() {
	log.Println("Stopping WebSocket server...")

	// Cancel context
	s.cancel()

	// Close all connections
	s.mu.Lock()
	for _, client := range s.connections {
		client.Close()
	}
	s.connections = make(map[string]*Client)
	s.mu.Unlock()

	// Shutdown HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}

	log.Println("WebSocket server stopped")
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Get user ID from query params or headers
	userID := r.URL.Query().Get("user_id")
	token := r.URL.Query().Get("token")

	// If no user_id in query, wait for connect message
	if userID == "" {
		// Create client without userID for now
		client := &Client{
			userID: "",
			conn:   conn,
			send:   make(chan []byte, 256),
			server: s,
		}

		// Start reader/writer goroutines
		go client.readPump()
		go client.writePump()

		// Wait for connect message to set userID
		return
	}

	// Validate token (in production, validate against auth service)
	if token == "" {
		log.Printf("WebSocket connection without token for user: %s", userID)
		conn.Close()
		return
	}

	// Register client
	client := &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
	}

	s.registerClient(client)

	// Send connection acknowledgment
	ackMsg := WebSocketMessage{
		Type:   MsgTypeAck,
		UserID: userID,
		Status: "connected",
	}
	ackData, _ := json.Marshal(ackMsg)
	client.send <- ackData

	// Start reader/writer goroutines
	go client.readPump()
	go client.writePump()

	log.Printf("WebSocket client connected: %s", userID)
}

// registerClient registers a client
func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close existing connection for this user if any
	if existing, ok := s.connections[client.userID]; ok {
		existing.Close()
	}

	s.connections[client.userID] = client
}

// unregisterClient unregisters a client
func (s *Server) unregisterClient(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.connections[userID]; ok {
		delete(s.connections, userID)
		client.Close()
		log.Printf("WebSocket client disconnected: %s", userID)
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		if c.userID != "" {
			c.server.unregisterClient(c.userID)
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(message []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return
	}

	switch msg.Type {
	case MsgTypeConnect:
		// Handle connect message
		if msg.UserID != "" && c.userID == "" {
			c.mu.Lock()
			c.userID = msg.UserID
			c.mu.Unlock()

			c.server.registerClient(c)

			// Send acknowledgment
			ackMsg := WebSocketMessage{
				Type:   MsgTypeAck,
				UserID: msg.UserID,
				Status: "connected",
			}
			ackData, _ := json.Marshal(ackMsg)
			c.send <- ackData

			log.Printf("WebSocket client authenticated: %s", msg.UserID)
		}

	case MsgTypePing:
		// Respond with pong
		pongMsg := WebSocketMessage{Type: MsgTypePong}
		pongData, _ := json.Marshal(pongMsg)
		c.send <- pongData
	}
}

// handleBroadcast handles broadcasting events to connected clients
func (s *Server) handleBroadcast() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.broadcast(event)
		}
	}
}

// broadcast broadcasts an event to specified users
func (s *Server) broadcast(event *BroadcastEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, userID := range event.UserIDs {
		if client, ok := s.connections[userID]; ok {
			msg := WebSocketMessage{
				Type:    event.Type,
				Payload: json.RawMessage(event.Payload.(string)),
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal broadcast message: %v", err)
				continue
			}
			select {
			case client.send <- data:
			default:
				log.Printf("Failed to send to client %s: channel full", userID)
			}
		}
	}
}

// SendBadgeEarned sends a badge earned event to a user
func (s *Server) SendBadgeEarned(userID, badgeID, badgeName, description string, points int) {
	payload := BadgeEarnedPayload{
		BadgeID:          badgeID,
		BadgeName:        badgeName,
		Description:      description,
		Points:           points,
		BadgeIcon:        "emoji_events",
		PointsAwarded:    points,
		BadgeDescription: description,
		EarnedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	payloadJSON, _ := json.Marshal(payload)

	event := &BroadcastEvent{
		UserIDs: []string{userID},
		Type:    MsgTypeBadgeEarned,
		Payload: string(payloadJSON),
	}

	select {
	case s.eventChan <- event:
	default:
		log.Printf("Failed to queue badge earned event: channel full")
	}
}

// SendPointsUpdated sends a points updated event to a user
func (s *Server) SendPointsUpdated(userID string, points int) {
	msg := WebSocketMessage{
		Type:   MsgTypePointsUpdated,
		UserID: userID,
		Points: points,
	}

	data, _ := json.Marshal(msg)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, ok := s.connections[userID]; ok {
		select {
		case client.send <- data:
		default:
			log.Printf("Failed to send points update: channel full")
		}
	}
}

// SendUserStats sends user stats to a user
func (s *Server) SendUserStats(userID string, stats interface{}) {
	statsJSON, _ := json.Marshal(stats)

	msg := WebSocketMessage{
		Type:  MsgTypeUserStats,
		Stats: statsJSON,
	}

	data, _ := json.Marshal(msg)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, ok := s.connections[userID]; ok {
		select {
		case client.send <- data:
		default:
			log.Printf("Failed to send user stats: channel full")
		}
	}
}

// SendStreakUpdated sends streak update to a user
func (s *Server) SendStreakUpdated(userID string, streak int) {
	msg := WebSocketMessage{
		Type:   MsgTypeStreakUpdated,
		UserID: userID,
		Streak: streak,
	}

	data, _ := json.Marshal(msg)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, ok := s.connections[userID]; ok {
		select {
		case client.send <- data:
		default:
			log.Printf("Failed to send streak update: channel full")
		}
	}
}

// BroadcastToUsers broadcasts an event to multiple users
func (s *Server) BroadcastToUsers(userIDs []string, eventType string, payload interface{}) {
	payloadJSON, _ := json.Marshal(payload)

	event := &BroadcastEvent{
		UserIDs: userIDs,
		Type:    eventType,
		Payload: string(payloadJSON),
	}

	select {
	case s.eventChan <- event:
	default:
		log.Printf("Failed to queue broadcast event: channel full")
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// GetConnectedUsers returns the list of connected user IDs
func (s *Server) GetConnectedUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]string, 0, len(s.connections))
	for userID := range s.connections {
		users = append(users, userID)
	}
	return users
}

// GetConnectionCount returns the number of active connections
func (s *Server) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}
