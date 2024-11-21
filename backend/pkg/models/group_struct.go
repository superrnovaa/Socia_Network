package models

import "time"

type Group struct {
	ID          int        `json:"id"`
	Name        string     `json:"title"`
	Description string     `json:"description"`
	CreatorID   int        `json:"creator_id"`
	ImageURL    string     `json:"image"`
	CreatedAt   time.Time  `json:"created_at"`
	Members     []UserItem `json:"members"`
	Type        string     `json:"type"` // New field to categorize the group
}

type GroupMember struct {
	ID        int       `json:"id"`
	GroupID   int       `json:"group_id"`
	UserID    int       `json:"user_id"`
	Status    string    `json:"status"` // "pending" or "accepted"
	CreatedAt time.Time `json:"created_at"`
}

type InviteUser struct {
	User   UserItem `json:"user"`
	Status string   `json:"status"` // "pending" or "accepted"
}

type Event struct {
	ID           int       `json:"id"`
	GroupID      int       `json:"group_id"`
	GroupName    string    `json:"group_name"`
	Creator      SafeUser  `json:"creator"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	EventDate    time.Time `json:"event_date"`
	CreatedAt    time.Time `json:"created_at"`
	UserResponse string    `json:"user_response"`
}

type EventResponse struct {
	ID       int      `json:"id"`
	EventID  int      `json:"event_id"`
	User     SafeUser `json:"user"`
	Response string   `json:"response"` // "going" or "not_going"
}
