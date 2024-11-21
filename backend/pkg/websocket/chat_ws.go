package websocket

import (
	query "backend/pkg/db/queries"
	"backend/pkg/models"
	"encoding/json"
	"log"
)

func SendChatToUsers(hub *Hub, chatMessage models.ChatMessage) {
	// Create the message to send
	message := Message{
		Type:    "chat",
		Payload: chatMessage,
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling chat: %v", err)
		return
	}

	// Find the client associated with the userID
	SendMessageToUser(hub, chatMessage.SenderID, payload)
	SendMessageToUser(hub, chatMessage.ReceiverID, payload)

	err = query.CreateChatNotifications(chatMessage.ID, []int{chatMessage.ReceiverID})
	if err != nil {
		log.Printf("Error sending Notifications: %v", err)
		return
	}
}

func SendGroupChatToUsers(hub *Hub, chatMessage models.ChatMessage) {
	// Create the message to send
	message := Message{
		Type:    "chat",
		Payload: chatMessage,
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling chat: %v", err)
		return
	}

	// Loop through group user IDs
	group, err := query.GetGroupData(chatMessage.GroupID)
	if err != nil {
		log.Printf("Error getting group members: %v", err)
		return
	}

	var groupIds []int
	for _, member := range group.Members {
		groupIds = append(groupIds, member.ID)
		SendMessageToUser(hub, member.ID, payload)
	}

	err = query.CreateChatNotifications(chatMessage.ID, groupIds)
	if err != nil {
		log.Printf("Error sending Notifications: %v", err)
		return
	}
}
