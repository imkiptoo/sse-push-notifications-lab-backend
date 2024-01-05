package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"sync"
	"time"
)

type userConnection struct {
	userID    string
	sendEvent chan []byte // Changed to a bidirectional channel
}

var (
	connections = struct {
		sync.RWMutex
		mapping map[string]chan []byte
	}{mapping: make(map[string]chan []byte)}
)

func main() {
	e := echo.New()
	e.HideBanner = true

	// SSE endpoint
	e.GET("/events", func(c echo.Context) error {
		userID := c.QueryParam("userID") // Assuming you pass userID from the client

		if connections.mapping[userID] != nil {
			return c.String(http.StatusBadRequest, "User already connected")
		}

		fmt.Printf("User %v connected \n\n", userID)

		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
		c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

		notify := c.Response().Writer

		// Create a channel for this user to handle notifications
		sendEvent := make(chan []byte, 10) // Buffered channel to handle send/receive
		userConn := userConnection{userID: userID, sendEvent: sendEvent}

		connections.Lock()
		connections.mapping[userID] = sendEvent
		connections.Unlock()

		defer func() {
			connections.Lock()
			delete(connections.mapping, userID) // Remove the user's channel when they disconnect
			connections.Unlock()
			close(sendEvent)
		}()

		go sendNotifications(&userConn)

		for {
			select {
			case event := <-sendEvent:
				_, err := fmt.Fprintf(notify, "data: %s\n\n", event)
				if err != nil {
					return err // Client disconnected
				}
				notify.(http.Flusher).Flush()
				/*case <-time.After(30 * time.Second):
				_, err := fmt.Fprintf(notify, ":\n\n") // Send a comment line as keep-alive
				if err != nil {
					return err // Client disconnected
				}
				notify.(http.Flusher).Flush()*/
			}
		}
	})

	err := e.Start(":8094")
	if err != nil {
		fmt.Println("Error starting server: ", err)
		return
	}
}

func sendNotifications(userConn *userConnection) {
	ticker := time.NewTicker(5 * time.Second) // Adjust this interval as needed
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Simulated notification; replace with your notification logic
			data := []byte("Hello " + userConn.userID + " at " + time.Now().String())
			userConn.sendEvent <- data // Send notification to the user's channel
			//fmt.Printf("%v: Sent notification to: [%v], content: [%v]\n\n", time.Now().String(), userConn.userID, string(data))
			fmt.Printf("%v: %v\n", userConn.userID, time.Now().Second())
		case <-userConn.sendEvent:
			// Close the goroutine when the user's channel is closed (user disconnected)
			return
		}
	}
}
