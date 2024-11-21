package websocket

import (
	"backend/pkg/models"
	"encoding/json"
	"log"
)

func SendNotificationToUser(hub *Hub, userID int, notification models.Notification) {
	// Create the message to send
	message := Message{
		Type:    "notification",
		Payload: notification,
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling notification: %v", err)
		return
	}

	// Find the client associated with the userID
	client, ok := hub.clients[userID]
	if !ok {
		log.Printf("User with ID %d not connected", userID)
		return
	}

	// Send the message to the specific user's WebSocket connection
	select {
	case client.send <- payload:
		log.Printf("Notification sent to user %d", userID)
	default:
		log.Printf("User %d's WebSocket connection is not ready", userID)
	}
}

func SendDeNotificationToUser(hub *Hub, userID int) {
	// Create the message to send
	message := Message{
		Type: "denotification",
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling notification: %v", err)
		return
	}

	// Find the client associated with the userID
	client, ok := hub.clients[userID]
	if !ok {
		log.Printf("User with ID %d not connected", userID)
		return
	}

	// Send the message to the specific user's WebSocket connection
	select {
	case client.send <- payload:
		log.Printf("Notification sent to user %d", userID)
	default:
		log.Printf("User %d's WebSocket connection is not ready", userID)
	}
}
