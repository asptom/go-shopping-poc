package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/logging"
	ws "go-shopping-poc/internal/platform/websocket"

	"github.com/gorilla/websocket"
)

func echoHandler(logger *slog.Logger, conn *websocket.Conn) {
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error("WebSocket read error", "error", err.Error())
			break
		}
		logger.Info("WebSocket received message", "message", string(msg))
		if err := conn.WriteMessage(mt, msg); err != nil {
			logger.Error("WebSocket write error", "error", err.Error())
			break
		}
	}
}

func main() {
	loggerProvider, err := logging.NewLoggerProvider(logging.LoggerConfig{
		ServiceName: "websocket",
	})
	if err != nil {
		slog.Error("Failed to create logger provider", "error", err.Error())
		os.Exit(1)
	}
	logger := loggerProvider.Logger()

	wsCfg, err := ws.LoadConfig()
	if err != nil {
		logger.Error("Failed to load WebSocket config", "error", err.Error())
		os.Exit(1)
	}

	wsServer, err := ws.NewWebSocketServer()
	if err != nil {
		logger.Error("Failed to create WebSocket server", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("WebSocket server starting")
	addr := wsCfg.Port
	logger.Info("WebSocket server listening", "address", addr+"/ws")

	httpServer := &http.Server{
		Addr:         addr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	http.HandleFunc("/ws", wsServer.Handle(func(conn *websocket.Conn) {
		echoHandler(logger, conn)
	}))

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("ListenAndServe error", "error", err.Error())
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("Shutting down WebSocket server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err.Error())
	}
}
