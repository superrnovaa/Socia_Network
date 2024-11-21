package models

import (
	"time"
)

type ChatMessage struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"senderId"`
	ReceiverID int       `json:"receiverId"`
	GroupID    int       `json:"groupId"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Chat struct {
	Messages     []ChatMessage `json:"messages"`
	UserA        UserItem      `json:"userA"`
	UserB        UserItem      `json:"userB"`
	Group        Group         `json:"group"`
	Notification int           `json:"notification"`
	AllowChat    bool          `json:"allowChat"`
}
