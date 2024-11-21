package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"fmt"
	"log"
)

func CreatePostQuery(newPost models.Post) (int64, error) {
	result, err := sqlite.DB.Exec(
		"INSERT INTO posts (title, content, user_id, image_url, privacy) VALUES (?, ?, ?, ?, ?)",
		newPost.Title,
		newPost.Content,
		newPost.User.ID,
		newPost.File,
		newPost.Privacy,
	)

	if err != nil {
		log.Printf("Error creating post: %v", err)
		return 0, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return 0, err
	}

	return postID, nil
}

func InsertPostViewer(postId int64, viewerId int) error {
	_, err := sqlite.DB.Exec(
		"INSERT INTO post_viewers (post_id, viewer_id) VALUES (?, ?)",
		postId,
		viewerId,
	)

	if err != nil {
		log.Printf("Error inserting post viewer: %v", err)
		return err
	}

	return nil
}

func GetUserIDFromPostID(postId int) (int, error) {
	var user int
	err := sqlite.DB.QueryRow("SELECT user_id FROM posts WHERE id = $1", postId).Scan(&user)
	if err != nil {
		return user, err
	}
	return user, nil
}

func GetPostsQuery(userID int) ([]models.Post, error) {
	query := `
		SELECT DISTINCT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at, 
			   u.id, u.username, u.avatar_url,
			   g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN groups g ON p.group_id = g.id
		LEFT JOIN followers f ON f.followed_id = p.user_id AND f.follower_id = ?
		LEFT JOIN group_members gm ON gm.group_id = p.group_id AND gm.user_id = ?
		LEFT JOIN post_viewers pv ON pv.post_id = p.id AND pv.viewer_id = ?
		WHERE (
			-- Posts from groups the user is a member of
			(p.group_id IS NOT NULL AND gm.status = 'accepted')
			OR
			-- Public posts from users with public profiles
			(p.privacy = 'public' AND u.is_public = TRUE AND p.group_id IS NULL)
			OR
			-- Almost private posts where the user is included
			(p.privacy = 'almost_private' AND pv.viewer_id IS NOT NULL AND p.group_id IS NULL)
			OR
			-- Private posts from users the current user is following (excluding group posts)
			(p.privacy = 'private' AND f.status = 'accepted' AND p.group_id IS NULL)
			OR
			-- User's own posts
			p.user_id = ?
		)
		ORDER BY p.created_at DESC
	`
	rows, err := sqlite.DB.Query(query, userID, userID, userID, userID)
	if err != nil {
		log.Printf("Error retrieving posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanPosts(rows)
}

func UpdatePostQuery(updatedPost models.Post) error {
	query := `
		UPDATE posts 
		SET title = ?, 
			content = ?, 
			privacy = ?,
			image_url = ?
		WHERE id = ?
	`
	_, err := sqlite.DB.Exec(
		query,
		updatedPost.Title,
		updatedPost.Content,
		updatedPost.Privacy,
		updatedPost.File,
		updatedPost.ID,
	)
	if err != nil {
		log.Printf("Error updating post: %v", err)
		return err
	}
	return nil
}

func DeletePostQuery(postID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		return err
	}
	return nil
}

// Add this function to the existing file
func GetSinglePostQuery(postID string, currentUserID int) (models.Post, error) {
	query := `
		SELECT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at, 
			   u.id, u.username, u.avatar_url,
			   g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN groups g ON p.group_id = g.id
		LEFT JOIN followers f ON f.followed_id = p.user_id AND f.follower_id = ?
		LEFT JOIN post_viewers pv ON pv.post_id = p.id AND pv.viewer_id = ?
		LEFT JOIN group_members gm ON gm.group_id = p.group_id AND gm.user_id = ?
		WHERE p.id = ? AND (
			p.user_id = ? OR
			(p.privacy = 'public' AND u.is_public = TRUE AND p.group_id IS NULL) OR
			(p.privacy = 'almost_private' AND pv.viewer_id IS NOT NULL AND p.group_id IS NULL) OR
			(p.privacy = 'private' AND f.status = 'accepted' AND p.group_id IS NULL) OR
			(p.group_id IS NOT NULL AND gm.status = 'accepted')
		)
	`
	var post models.Post
	var safeUser models.SafeUser
	var imageURL sql.NullString
	var groupID, groupName, groupDescription sql.NullString
	var groupCreatorID sql.NullInt64
	var groupImageURL sql.NullString
	var groupCreatedAt sql.NullTime

	err := sqlite.DB.QueryRow(query, currentUserID, currentUserID, currentUserID, postID, currentUserID).Scan(
		&post.ID, &post.Title, &post.Content, &imageURL, &post.Privacy, &post.CreatedAt,
		&safeUser.ID, &safeUser.Username, &safeUser.AvatarURL,
		&groupID, &groupName, &groupDescription, &groupCreatorID, &groupImageURL, &groupCreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Post{}, fmt.Errorf("post not found or user not permitted to view")
		}
		log.Printf("Error retrieving single post: %v", err)
		return models.Post{}, err
	}

	if imageURL.Valid {
		post.File = imageURL.String
	} else {
		post.File = ""
	}
	post.User = safeUser

	// Fetch comments separately
	commentsQuery := `
		SELECT c.id, c.content, c.created_at, u.id, u.username, u.avatar_url
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at DESC
	`
	rows, err := sqlite.DB.Query(commentsQuery, postID)
	if err != nil {
		log.Printf("Error retrieving comments for post: %v", err)
		return post, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		var commentUser models.SafeUser
		err := rows.Scan(
			&comment.ID, &comment.Content, &comment.CreatedAt,
			&commentUser.ID, &commentUser.Username, &commentUser.AvatarURL,
		)
		if err != nil {
			log.Printf("Error scanning comment row: %v", err)
			return post, err
		}
		comment.User = commentUser
		comments = append(comments, comment)
	}

	post.Comments = comments
	return post, nil
}

func GetFollowedPostsQuery(userID int) ([]models.Post, error) {
	query := `
        SELECT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at, 
               u.id AS user_id, u.username, u.avatar_url, p.created_at AS order_date,
               g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN groups g ON p.group_id = g.id
        LEFT JOIN group_members gm ON gm.group_id = p.group_id AND gm.user_id = ?
        LEFT JOIN followers f ON f.followed_id = p.user_id AND f.follower_id = ?
        LEFT JOIN post_viewers pv ON pv.post_id = p.id AND pv.viewer_id = ?
        WHERE 
            (p.user_id = ?) OR  -- User's own posts
            (f.status = 'accepted' AND (
                (p.privacy = 'public' AND p.group_id IS NULL) OR
                (p.privacy = 'private' AND p.group_id IS NULL) OR
                (p.privacy = 'almost_private' AND pv.viewer_id IS NOT NULL AND p.group_id IS NULL) OR
                (p.group_id IS NOT NULL AND gm.status = 'accepted')
            ))
        ORDER BY p.created_at DESC
    `
	rows, err := sqlite.DB.Query(query, userID, userID, userID, userID)
	if err != nil {
		log.Printf("Error retrieving followed posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanPosts(rows)
}

func GetUserPostsQuery(targetUserID, currentUserID int) ([]models.Post, error) {
	query := `
		SELECT DISTINCT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at, 
			   u.id, u.username, u.avatar_url,
			   g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN groups g ON p.group_id = g.id
		LEFT JOIN followers f ON f.followed_id = p.user_id AND f.follower_id = ?
		LEFT JOIN post_viewers pv ON pv.post_id = p.id AND pv.viewer_id = ?
		LEFT JOIN group_members gm ON gm.group_id = p.group_id AND gm.user_id = ?
		WHERE p.user_id = ? AND (
			-- Public posts if the current user is following the target user
			(p.privacy = 'public' AND (u.is_public = TRUE OR f.status = 'accepted') AND p.group_id IS NULL)
			OR
			-- Almost private posts where the current user is included
			(p.privacy = 'almost_private' AND pv.viewer_id IS NOT NULL AND p.group_id IS NULL)
			OR
			-- Private posts if the current user is following the target user
			(p.privacy = 'private' AND f.status = 'accepted' AND p.group_id IS NULL)
			OR
			-- Group posts where both users are members
			(p.group_id IS NOT NULL AND gm.status = 'accepted')
			OR
			-- All posts if the current user is viewing their own profile
			? = ?
		)
		ORDER BY p.created_at DESC
	`
	rows, err := sqlite.DB.Query(query, currentUserID, currentUserID, currentUserID, targetUserID, currentUserID, targetUserID)
	if err != nil {
		log.Printf("Error retrieving user posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanPosts(rows)
}

func scanPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var safeUser models.SafeUser
		var imageURL sql.NullString
		var groupID, groupCreatorID sql.NullInt64
		var groupName, groupDescription, groupImageURL sql.NullString
		var groupCreatedAt sql.NullTime

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &imageURL, &p.Privacy, &p.CreatedAt,
			&safeUser.ID, &safeUser.Username, &safeUser.AvatarURL,
			&groupID, &groupName, &groupDescription, &groupCreatorID, &groupImageURL, &groupCreatedAt,
		); err != nil {
			log.Printf("Error scanning post row: %v", err)
			return nil, err
		}

		if imageURL.Valid {
			p.File = imageURL.String
		}

		if groupID.Valid {
			p.Group = &models.Group{
				ID:          int(groupID.Int64),
				Name:        groupName.String,
				Description: groupDescription.String,
				CreatorID:   int(groupCreatorID.Int64),
				ImageURL:    groupImageURL.String,
				CreatedAt:   groupCreatedAt.Time,
			}
		}

		p.User = safeUser
		posts = append(posts, p)
	}
	return posts, nil
}

func IsUserPermittedToViewPost(postID int, userID int) (bool, error) {
	query := `
		SELECT CASE
			WHEN p.user_id = ? THEN TRUE
			WHEN p.privacy = 'public' AND u.is_public = TRUE AND p.group_id IS NULL THEN TRUE
			WHEN p.privacy = 'private' AND EXISTS (
				SELECT 1 FROM followers WHERE follower_id = ? AND followed_id = p.user_id AND status = 'accepted'
			) AND p.group_id IS NULL THEN TRUE
			WHEN p.privacy = 'almost_private' AND EXISTS (
				SELECT 1 FROM post_viewers WHERE post_id = p.id AND viewer_id = ?
			) AND p.group_id IS NULL THEN TRUE
			WHEN p.group_id IS NOT NULL AND EXISTS (
				SELECT 1 FROM group_members WHERE group_id = p.group_id AND user_id = ? AND status = 'accepted'
			) THEN TRUE
			ELSE FALSE
		END AS is_permitted
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`
	var isPermitted bool
	err := sqlite.DB.QueryRow(query, userID, userID, userID, userID, postID).Scan(&isPermitted)
	if err != nil {
		return false, fmt.Errorf("error checking post permissions: %w", err)
	}
	return isPermitted, nil
}

func UpdatePostViewers(postID int, viewerIDs []int) error {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing viewers
	_, err = tx.Exec("DELETE FROM post_viewers WHERE post_id = ?", postID)
	if err != nil {
		return err
	}

	// Insert new viewers
	for _, viewerID := range viewerIDs {
		_, err = tx.Exec("INSERT INTO post_viewers (post_id, viewer_id) VALUES (?, ?)", postID, viewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Add this new function
func CreateGroupPostQuery(newPost models.Post) (int64, error) {
	result, err := sqlite.DB.Exec(
		"INSERT INTO posts (title, content, user_id, image_url, privacy, group_id) VALUES (?, ?, ?, ?, ?, ?)",
		newPost.Title,
		newPost.Content,
		newPost.User.ID,
		newPost.File,
		newPost.Privacy,
		newPost.Group.ID,
	)

	if err != nil {
		log.Printf("Error creating group post: %v", err)
		return 0, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return 0, err
	}

	return postID, nil
}

// Add this function to fetch group posts
func GetGroupPostsQuery(groupID int) ([]models.Post, error) {
	query := `
		SELECT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at, 
			   u.id, u.username, u.avatar_url,
			   g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN groups g ON p.group_id = g.id
		WHERE p.group_id = ?
		ORDER BY p.created_at DESC
	`
	rows, err := sqlite.DB.Query(query, groupID)
	if err != nil {
		log.Printf("Error retrieving group posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanPosts(rows)
}


