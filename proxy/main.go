package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

func proxyHandler(c echo.Context) error {
	// Upgrade the HTTP connection to a WebSocket connection
	clientConn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return err
	}
	defer clientConn.Close()

	// Connect to the target WebSocket server
	targetURL := "ws://localhost:1323/ws"
	targetConn, _, err := websocket.DefaultDialer.Dial(targetURL, nil)
	if err != nil {
		log.Println("Failed to connect to target WebSocket server:", err)
		return err
	}
	defer targetConn.Close()

	// Channel to signal the closure of the connections
	done := make(chan struct{})

	// Forward messages from client to target server
	go func() {
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				log.Println("Error reading from client:", err)
				break
			}
			err = targetConn.WriteMessage(messageType, message)
			if err != nil {
				log.Println("Error writing to target server:", err)
				break
			}
		}
		done <- struct{}{}
	}()

	// Forward messages from target server to client
	go func() {
		for {
			messageType, message, err := targetConn.ReadMessage()
			if err != nil {
				log.Println("Error reading from target server:", err)
				break
			}
			err = clientConn.WriteMessage(messageType, message)
			if err != nil {
				log.Println("Error writing to client:", err)
				break
			}
		}
		done <- struct{}{}
	}()

	// Wait for either connection to close
	<-done

	return nil
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/ws", func(c echo.Context) error {
		return proxyHandler(c)
	})

	log.Println("Starting WebSocket proxy server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
