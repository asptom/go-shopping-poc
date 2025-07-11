package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/logging"
	ws "go-shopping-poc/pkg/websocket"

	"github.com/gorilla/websocket"
)

func echoHandler(conn *websocket.Conn) {
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			logging.Error("Read error: %v", err)
			break
		}
		logging.Info("Received: %s", msg)
		// Example business logic: echo back
		if err := conn.WriteMessage(mt, msg); err != nil {
			logging.Error("Write error: %v", err)
			break
		}
	}
}

func main() {

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Optionally set log level from env/config
	logging.SetLevel("DEBUG")

	// Create WebSocket server
	server := ws.NewWebSocketServer(cfg)

	http.HandleFunc("/ws", server.Handle(echoHandler))

	logging.Info("The WebSocket server is starting...")
	//addr := ":80"
	addr := cfg.WebSocketPort()
	logging.Info("WebSocket server listening on %s/ws", addr)

	// Start server in a goroutine
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logging.Error("ListenAndServe: %v", err)
		}
	}()

	// Wait for interrupt to exit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Info("Shutting down WebSocket server...")
}
