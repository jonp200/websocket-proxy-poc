package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var allowedOrigins = []string{"*"}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isOriginAllowed(allowedOrigins, origin)
	},
}

func isOriginAllowed(allowedOrigins []string, origin string) bool {
	for _, o := range allowedOrigins {
		// If wild-card appears at least once, all origins will be allowed
		if o == "*" {
			return true
		}
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
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
	done := make(chan bool)

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
		done <- true
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
		done <- true
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
