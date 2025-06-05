package websocket

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// mockConfig implements only the methods needed for NewWebSocketServer.
type mockConfig struct{}

func (m *mockConfig) WebSocketReadBufferSize() int  { return 1024 }
func (m *mockConfig) WebSocketWriteBufferSize() int { return 1024 }

// TestWebSocketServer_Handle tests the WebSocketServer.Handle method.
func TestWebSocketServer_Handle(t *testing.T) {
	// Use mockConfig for buffer sizes
	cfg := &mockConfig{}

	// Create server using only the buffer size methods
	server := &WebSocketServer{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.WebSocketReadBufferSize(),
			WriteBufferSize: cfg.WebSocketWriteBufferSize(),
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Clients: make(map[*websocket.Conn]bool),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Handler echoes "hello" -> "world"
	handler := server.Handle(func(conn *websocket.Conn) {
		defer wg.Done()
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("ReadMessage error: %v", err)
			return
		}
		if mt != websocket.TextMessage || string(msg) != "hello" {
			t.Errorf("Unexpected message: type=%v msg=%s", mt, msg)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("world")); err != nil {
			t.Errorf("WriteMessage error: %v", err)
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer ws.Close()

	if err := ws.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("WriteMessage error: %v", err)
	}

	mt, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error: %v", err)
	}
	if mt != websocket.TextMessage || string(msg) != "world" {
		t.Errorf("Unexpected response: type=%v msg=%s", mt, msg)
	}

	wg.Wait()
}
