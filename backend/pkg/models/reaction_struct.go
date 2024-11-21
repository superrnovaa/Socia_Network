package models

import "time"

// Update the existing Reaction struct
type Reaction struct {
	ID             int       `json:"id"`
	PostID         *int      `json:"post_id"`    // Pointer to allow NULL
	CommentID      *int      `json:"comment_id"` // Pointer to allow NULL
	UserID         int       `json:"user_id"`
	ReactionTypeID int       `json:"reaction_type_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// Add the new ReactionType struct
type ReactionType struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}
