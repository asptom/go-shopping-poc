package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	ws "go-shopping-poc/internal/platform/websocket"

	"github.com/gorilla/websocket"
)

func echoHandler(conn *websocket.Conn) {
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ERROR] Websocket: Read error: %v", err)
			break
		}
		log.Printf("[INFO] Websocket: Received: %s", msg)
		// Example business logic: echo back
		if err := conn.WriteMessage(mt, msg); err != nil {
			log.Printf("[ERROR] Websocket: Write error: %v", err)
			break
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags)

	// Load platform WebSocket configuration
	wsCfg, err := ws.LoadConfig()
	if err != nil {
		log.Printf("[ERROR] Websocket: Failed to load WebSocket config: %v", err)
		os.Exit(1)
	}

	server, err := ws.NewWebSocketServer()
	if err != nil {
		log.Printf("[ERROR] Websocket: Failed to create WebSocket server: %v", err)
		os.Exit(1)
	}

	http.HandleFunc("/ws", server.Handle(echoHandler))

	log.Printf("[DEBUG] Websocket: The WebSocket server is starting...")
	addr := wsCfg.Port
	log.Printf("[DEBUG] Websocket: server listening on %s/ws", addr)

	// Start server in a goroutine
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("[ERROR] Websocket: ListenAndServe: %v", err)
		}
	}()

	// Wait for interrupt to exit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("[INFO] Websocket: Shutting down WebSocket server...")
}
