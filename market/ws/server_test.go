package ws

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tent-of-trials/market/matching"
	"github.com/tent-of-trials/market/orderbook"
	"github.com/tent-of-trials/market/types"
	"go.uber.org/zap"
)

func TestWebSocketHeartbeatDisconnectsIdleClient(t *testing.T) {
	os.Setenv("WS_HEARTBEAT_INTERVAL_SECS", "1")
	defer os.Unsetenv("WS_HEARTBEAT_INTERVAL_SECS")

	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	go hub.Run()

	config := matching.EngineConfig{}
	books := map[types.Symbol]*orderbook.OrderBook{}
	engine := matching.NewMatchingEngine(config, books)

	server := NewServer(hub, engine, logger, 0)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("could not connect: %v", err)
	}

	// Disable automatic pong responses to simulate an idle client
	conn.SetPingHandler(func(appData string) error {
		return nil
	})

	// Try reading to see if it closes.
	done := make(chan bool)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("connection was not closed after 2 heartbeat intervals")
	case <-done:
		// success, connection closed
	}
}

func TestWebSocketHeartbeatKeepsActiveClientAlive(t *testing.T) {
	os.Setenv("WS_HEARTBEAT_INTERVAL_SECS", "1")
	defer os.Unsetenv("WS_HEARTBEAT_INTERVAL_SECS")

	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	go hub.Run()

	config := matching.EngineConfig{}
	books := map[types.Symbol]*orderbook.OrderBook{}
	engine := matching.NewMatchingEngine(config, books)

	server := NewServer(hub, engine, logger, 0)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Setup pong handler manually for client, though the default gorilla client 
	// does not automatically respond to pings unless we do this loop:
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	done := make(chan bool)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				done <- true
				return
			}
		}
	}()

	// Wait 3 seconds, the connection should NOT be closed because we are sending pongs.
	select {
	case <-time.After(3 * time.Second):
		// success, connection is still alive
	case <-done:
		t.Fatal("connection was unexpectedly closed despite sending pongs")
	}
}
