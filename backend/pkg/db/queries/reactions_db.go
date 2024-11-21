package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"fmt"
)

// GetAvailableReactions fetches all reactions from the reaction_types table
func GetAvailableReactions() ([]models.ReactionType, error) {
	query := `SELECT id, name, icon_url FROM reaction_types`
	rows, err := sqlite.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying reaction types: %v", err)
	}
	defer rows.Close()

	var reactions []models.ReactionType
	for rows.Next() {
		var r models.ReactionType
		if err := rows.Scan(&r.ID, &r.Name, &r.IconURL); err != nil {
			return nil, fmt.Errorf("error scanning reaction type: %v", err)
		}
		reactions = append(reactions, r)
	}

	return reactions, nil
}

// AddOrUpdateReaction adds a new reaction, updates an existing one, or removes it if it's the same
func AddOrUpdateReaction(reaction models.Reaction) (bool, error) {
	// First, check if the reaction already exists
	existingReaction, err := GetUserReaction(reaction.UserID, reaction.PostID, reaction.CommentID)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("error checking existing reaction: %v", err)
	}

	if existingReaction != nil && existingReaction.ReactionTypeID == reaction.ReactionTypeID {
		// If the reaction is the same, remove it
		query := `DELETE FROM reactions WHERE user_id = ? AND (post_id = ? OR comment_id = ?)`
		_, err := sqlite.DB.Exec(query, reaction.UserID, reaction.PostID, reaction.CommentID)
		if err != nil {
			return false, fmt.Errorf("error removing reaction: %v", err)
		}
		return true, nil // Return true indicating the reaction was the same and removed
	}

	// If the reaction is different or doesn't exist, add or update it
	query := `
		INSERT INTO reactions (post_id, comment_id, user_id, reaction_type_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, IFNULL(post_id, 0), IFNULL(comment_id, 0))
		DO UPDATE SET reaction_type_id = excluded.reaction_type_id
	`
	_, err = sqlite.DB.Exec(query, reaction.PostID, reaction.CommentID, reaction.UserID, reaction.ReactionTypeID)
	if err != nil {
		return false, fmt.Errorf("error adding or updating reaction: %v", err)
	}
	return false, nil // Return false indicating the reaction was different or added/updated
}

// GetReactionsByContent retrieves reactions for a post or comment
func GetReactionsByContent(postID, commentID *int) (map[string]int, error) {
	query := `
		SELECT rt.name, COUNT(r.id)
		FROM reactions r
		JOIN reaction_types rt ON r.reaction_type_id = rt.id
		WHERE (r.post_id = ? OR r.comment_id = ?)
		GROUP BY rt.name
	`
	rows, err := sqlite.DB.Query(query, postID, commentID)
	if err != nil {
		return nil, fmt.Errorf("error querying reactions: %v", err)
	}
	defer rows.Close()

	reactions := make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("error scanning reaction count: %v", err)
		}
		reactions[name] = count
	}

	return reactions, nil
}

// GetUserReaction retrieves a user's reaction to a post or comment
func GetUserReaction(userID int, postID, commentID *int) (*models.Reaction, error) {
	query := `
		SELECT id, post_id, comment_id, user_id, reaction_type_id, created_at
		FROM reactions
		WHERE user_id = ? AND (post_id = ? OR comment_id = ?)
	`
	var reaction models.Reaction
	err := sqlite.DB.QueryRow(query, userID, postID, commentID).Scan(
		&reaction.ID, &reaction.PostID, &reaction.CommentID,
		&reaction.UserID, &reaction.ReactionTypeID, &reaction.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error querying user reaction: %v", err)
	}
	return &reaction, nil
}
