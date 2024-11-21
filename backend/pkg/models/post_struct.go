package models

import "time"

type Post struct {
	ID           int            `json:"id"`
	User         SafeUser       `json:"user"`
	Title        string         `json:"title"`
	Content      string         `json:"content"`
	File         string         `json:"file"`
	Privacy      string         `json:"privacy"`
	CreatedAt    time.Time      `json:"created_at"`
	Comments     []Comment      `json:"comments,omitempty"`
	Reactions    map[string]int `json:"reactions"`
	UserReaction *Reaction      `json:"user_reaction"`
	Group        *Group         `json:"group,omitempty"` // Change this line
}

type SafeUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
}

type Privacy struct {
	Type    string `json:"type"`               // "private" or "almost-private"
	UserIDs []int  `json:"user_ids,omitempty"` // Slice of user IDs for "almost-private"
}

type Comment struct {
	ID           int            `json:"id"`
	PostID       int            `json:"post_id"`
	User         SafeUser       `json:"user"`
	File         string         `json:"file"`
	Content      string         `json:"content"`
	CreatedAt    time.Time      `json:"created_at"`
	Reactions    map[string]int `json:"reactions"`
	UserReaction *Reaction      `json:"user_reaction,omitempty"`
}
