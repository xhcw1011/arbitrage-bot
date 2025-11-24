package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"arbitrage-bot/pkg/ws"
)

func main() {
	log.Println("Testing EdgeX WebSocket...")

	// Create WebSocket client
	wsClient := ws.NewEdgeXWSClient("wss://quote.edgex.exchange/api/v1/public/ws")

	// Connect
	ctx := context.Background()
	if err := wsClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	// Wait for connection to establish
	time.Sleep(2 * time.Second)

	// Subscribe to BTC ticker (contractId: 10000001)
	err := wsClient.Subscribe("ticker.10000001", func(data json.RawMessage) {
		log.Printf("Received BTC ticker data: %s", string(data))
	})
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
	}

	// Subscribe to ETH ticker (contractId: 10000002)
	err = wsClient.Subscribe("ticker.10000002", func(data json.RawMessage) {
		log.Printf("Received ETH ticker data: %s", string(data))
	})
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
	}

	log.Println("Subscribed to tickers. Press Ctrl+C to exit...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
}
