package websocket

import (
	"net/http"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/logging"

	"github.com/gorilla/websocket"
)

// WebSocketClient wraps a websocket connection.
type WebSocketClient struct {
	conn *websocket.Conn
}

// NewWebSocketClient opens a websocket connection using the URL from config.
func NewWebSocketClient(cfg *config.Config) (*WebSocketClient, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: cfg.WebSocketTimeout(),
	}
	conn, _, err := dialer.Dial(cfg.GetWebSocketURL(), nil)
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

// NewWebSocketServer creates a new WebSocketServer with config.
func NewWebSocketServer(cfg *config.Config) *WebSocketServer {
	return &WebSocketServer{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.WebSocketReadBufferSize(),
			WriteBufferSize: cfg.WebSocketWriteBufferSize(),
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Clients: make(map[*websocket.Conn]bool),
	}
}

// HandlerFunc defines the signature for custom connection handlers.
type HandlerFunc func(conn *websocket.Conn)

// Handle handles websocket upgrade and connection, then calls the custom handler.
func (s *WebSocketServer) Handle(customHandler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			logging.Error("Upgrade error: %v", err)
			return
		}
		s.Clients[conn] = true
		go func() {
			defer func() {
				conn.Close()
				delete(s.Clients, conn)
			}()
			customHandler(conn)
		}()
	}
}
