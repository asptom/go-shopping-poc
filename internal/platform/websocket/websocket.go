package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocketClient wraps a websocket connection.
type WebSocketClient struct {
	conn *websocket.Conn
}

// NewWebSocketClient opens a websocket connection using the platform WebSocket config.
func NewWebSocketClient() (*WebSocketClient, error) {
	wsCfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: wsCfg.Timeout,
	}
	conn, _, err := dialer.Dial(wsCfg.URL, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketClient{conn: conn}, nil
}

// ReadMessage reads a message from the websocket.
func (c *WebSocketClient) ReadMessage() (messageType int, p []byte, err error) {
	return c.conn.ReadMessage()
}

// WriteMessage writes a message to the websocket.
func (c *WebSocketClient) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

// Close closes the websocket connection.
func (c *WebSocketClient) Close() error {
	return c.conn.Close()
}

// WebSocketServer wraps a websocket upgrader and manages connections.
type WebSocketServer struct {
	Upgrader websocket.Upgrader
	Clients  map[*websocket.Conn]bool
}

// createCheckOrigin creates a CheckOrigin function that validates against allowed origins.
func createCheckOrigin(allowedOrigins []string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	}
}

// NewWebSocketServer creates a new WebSocketServer with platform config.
func NewWebSocketServer() (*WebSocketServer, error) {
	wsCfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return &WebSocketServer{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  wsCfg.ReadBuffer,
			WriteBufferSize: wsCfg.WriteBuffer,
			CheckOrigin:     createCheckOrigin(wsCfg.AllowedOrigins),
		},
		Clients: make(map[*websocket.Conn]bool),
	}, nil
}

// HandlerFunc defines the signature for custom connection handlers.
type HandlerFunc func(conn *websocket.Conn)

// Handle handles websocket upgrade and connection, then calls the custom handler.
func (s *WebSocketServer) Handle(customHandler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR] Upgrade error: %v", err)
			return
		}
		s.Clients[conn] = true
		go func() {
			defer func() {
				_ = conn.Close()
				delete(s.Clients, conn)
			}()
			customHandler(conn)
		}()
	}
}
