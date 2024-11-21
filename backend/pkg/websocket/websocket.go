package websocket

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID int
}

// Message represents a generic message structure for WebSocket communication.
type Message struct {
	Type    string      `json:"type"`    // Type of the message (e.g., "notification")
	Payload interface{} `json:"payload"` // The actual data being sent (can be any type)
}

const MaxMessageLength = 2000

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Add message length validation
		if len(message) > MaxMessageLength {
			log.Printf("Message exceeds maximum length of %d bytes", MaxMessageLength)
			continue
		}

		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Extract userID from the query parameters
	userIDStr := r.URL.Query().Get("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Println("Invalid userID:", err)
		return
	}

	// Create a new Client instance with the userID
	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID, // Set the userID for the client
	}

	// Register the client in the hub
	hub.register <- client

	// Start the read and write pumps
	go client.writePump()
	go client.readPump()
}

func SendMessageToUser(hub *Hub, userID int, payload []byte) {
	client, ok := hub.clients[userID]
	if !ok {
		log.Printf("User with ID %d not connected", userID)
		return
	}
	// Send the message to the specific user's WebSocket connection
	select {
	case client.send <- payload: // Assuming client has a send channel
		log.Printf("Payload sent to user %d", userID)
	default:
		log.Printf("User %d's WebSocket connection is not ready", userID)
	}
}
