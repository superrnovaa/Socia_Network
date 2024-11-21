package models

import (
	"time"
)

type Notification struct {
	ID             int       `json:"id"`
	NotifiedUserID     int   `json:"notifiedUserId"`   
	NotifyingUserId int      `json:"notifyingUserId"` 
	Type           string    `json:"type"`      
	Content        string    `json:"content"`   
	IsRead         bool      `json:"isRead"`    
	CreatedAt      time.Time `json:"createdAt"`  
	NotifyingImage  string    `json:"notifyingImage"` 
	Object         string       `json:"object"` 
	ObjectID       int         `json:"objectId"` 
}
