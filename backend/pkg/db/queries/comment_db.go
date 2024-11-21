package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"time"

	"log"
)

func AddCommentQuery(newComment models.Comment) (models.Comment, error) {
	// Determine the file value to insert
	fileValue := newComment.File
	if fileValue == "" {
		fileValue = "" // Use an empty string if no file is provided
	}

	result, err := sqlite.DB.Exec("INSERT INTO comments (post_id, user_id, content, file) VALUES (?, ?, ?, ?)",
		newComment.PostID, newComment.User.ID, newComment.Content, fileValue) // Use fileValue
	if err != nil {
		log.Printf("Error adding comment: %v", err)
		return models.Comment{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		return models.Comment{}, err
	}

	newComment.ID = int(id)
	newComment.CreatedAt = time.Now()

	// Fetch the complete comment data
	comment, err := GetCommentByID(int(id))
	if err != nil {
		log.Printf("Error fetching newly created comment: %v", err)
		return newComment, nil
	}

	return comment, nil
}

func GetCommentByID(commentID int) (models.Comment, error) {
	query := `
		SELECT c.id, c.post_id, c.content, c.created_at, c.file, u.id, u.username, u.avatar_url
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = ?
	`
	var comment models.Comment
	var safeUser models.SafeUser
	err := sqlite.DB.QueryRow(query, commentID).Scan(
		&comment.ID, &comment.PostID, &comment.Content, &comment.CreatedAt,
		&comment.File, // Scan into sql.NullString
		&safeUser.ID, &safeUser.Username, &safeUser.AvatarURL,
	)
	if err != nil {
		log.Printf("Error fetching comment by ID: %v", err)
		return models.Comment{}, err
	}
	comment.User = safeUser
	return comment, nil
}

func GetCommentsQuery(postID string) ([]models.Comment, error) {
	rows, err := sqlite.DB.Query(`
		SELECT c.id, c.post_id, c.content, c.created_at, c.file, u.id, u.username, u.avatar_url
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at DESC
	`, postID)
	if err != nil {
		log.Printf("Error retrieving comments: %v", err)
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		var safeUser models.SafeUser
		if err := rows.Scan(&c.ID, &c.PostID, &c.Content, &c.CreatedAt, &c.File, // Include the file
			&safeUser.ID, &safeUser.Username, &safeUser.AvatarURL); err != nil {
			log.Printf("Error scanning comment row: %v", err)
			return nil, err
		}
		c.User = safeUser
		comments = append(comments, c)
	}

	return comments, nil
}

func UpdateCommentQuery(updatedComment models.Comment) error {
	_, err := sqlite.DB.Exec("UPDATE comments SET content = ?, file = ? WHERE id = ?", updatedComment.Content, updatedComment.File, updatedComment.ID)
	if err != nil {
		log.Printf("Error updating comment: %v", err)
		return err
	}
	return nil
}

func DeleteCommentQuery(commentID string) error {
	_, err := sqlite.DB.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		log.Printf("Error deleting comment: %v", err)
		return err
	}
	return nil
}
