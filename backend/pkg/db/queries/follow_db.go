package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"fmt"
	"log"
)

// FollowUser adds a follow relationship to the database
func FollowUser(followerID int, followedID int, status string) error {
	// Prepare the SQL statement
	query := `
		INSERT INTO followers (follower_id, followed_id, status)
		VALUES (?, ?, ?)
		ON CONFLICT(follower_id, followed_id) DO UPDATE SET status = excluded.status;`

	// Execute the statement
	_, err := sqlite.DB.Exec(query, followerID, followedID, status)
	if err != nil {
		return fmt.Errorf("failed to add follow relationship: %w", err)
	}

	return nil
}

// UnfollowUser removes a follow relationship from the database
func UnfollowUser(followerID int, followedID int) error {
	// Prepare the SQL statement
	query := `
		DELETE FROM followers
		WHERE follower_id = ? AND followed_id = ?;`

	// Execute the statement
	result, err := sqlite.DB.Exec(query, followerID, followedID)
	if err != nil {
		return fmt.Errorf("failed to unfollow user: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no follow relationship found to remove")
	}

	return nil
}

func GetUserStats(userID int) (int, int, error) {
	var Following int
	var Followers int
	// Query to count followers
	followersQuery := `
		SELECT COUNT(*) FROM followers WHERE followed_id = ? AND status = 'accepted';`
	err := sqlite.DB.QueryRow(followersQuery, userID).Scan(&Followers)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count followers: %w", err)
	}

	// Query to count following
	followingQuery := `
		SELECT COUNT(*) FROM followers WHERE follower_id = ? AND status = 'accepted';`
	err = sqlite.DB.QueryRow(followingQuery, userID).Scan(&Following)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count following: %w", err)
	}

	return Following, Followers, nil
}

// CheckIfUserFollows checks the follow relationship and returns the appropriate status
func CheckIfUserFollows(followerID int, followedID int) (string, error) {
	var status string

	query := `
		SELECT status 
		FROM followers 
		WHERE follower_id = $1 AND followed_id = $2
	`
	err := sqlite.DB.QueryRow(query, followerID, followedID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			// No follow relationship exists
			return "Follow", nil
		}
		return "", fmt.Errorf("error checking follow relationship: %w", err)
	}

	switch status {
	case "pending":
		return "Pending", nil
	case "accepted":
		return "Following", nil
	default:
		return "Following", nil // Default case if status is unknown
	}
}

func CheckIfUsersFollowsOrFollowed(userA int, userB int) (bool, error) {
	var status string

	fmt.Println("1")
	query := `
		SELECT status 
		FROM followers 
		WHERE (follower_id = $1 AND followed_id = $2) OR (follower_id = $2 AND followed_id = $1)
	`
	err := sqlite.DB.QueryRow(query, userA, userB).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			// No follow relationship exists
			fmt.Println("2")
			return false, nil
		}
		return false, fmt.Errorf("error checking follow relationship: %w", err)
	}
	fmt.Println(status)
	if status == "accepted" {
		return true, nil
	}
	fmt.Println(status)
	return false, nil // Default case if status is unknown
}

// GetFollowing retrieves the users that the specified user is following
func GetFollowing(userID int) ([]int, error) {
	rows, err := sqlite.DB.Query("SELECT followed_id FROM followers WHERE follower_id = ? AND status = 'accepted'", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var following []int
	for rows.Next() {
		var followedID int
		if err := rows.Scan(&followedID); err != nil {
			return nil, err
		}
		following = append(following, followedID)
	}
	return following, nil
}

// GetFollowers retrieves the users that are following the specified user
func GetFollowers(userID int) ([]int, error) {
	rows, err := sqlite.DB.Query("SELECT follower_id FROM followers WHERE followed_id = ? AND status = 'accepted'", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followers []int
	for rows.Next() {
		var followerID int
		if err := rows.Scan(&followerID); err != nil {
			return nil, err
		}
		followers = append(followers, followerID)
	}
	return followers, nil
}

// GetFollowersDetails retrieves the followers' IDs, usernames, and profile images for a specific user
func GetFollowersDetails(userID int) ([]models.UserItem, error) {
	var followers []models.UserItem

	query := `
		SELECT u.id, u.username, u.avatar_url
		FROM followers f
		JOIN users u ON f.follower_id = u.id
		WHERE f.followed_id = ? AND f.status = 'accepted';
	`

	rows, err := sqlite.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var follower models.UserItem
		if err := rows.Scan(&follower.ID, &follower.Username, &follower.ProfileImg); err != nil {
			return nil, err
		}
		// if profile img is null, set it to a default image
		if follower.ProfileImg == "" {
			follower.ProfileImg = "profileImage.png"
		}
		followers = append(followers, follower)
	}

	return followers, nil
}

// GetFollowersDetails retrieves the followers' IDs, usernames, and profile images for a specific user
func GetFollowingDetails(userID int) ([]models.UserItem, error) {
	var following []models.UserItem

	query := `
		SELECT u.id, u.username, u.avatar_url
		FROM followers f
		JOIN users u ON f.followed_id = u.id
		WHERE f.follower_id = ? AND f.status = 'accepted';
	`

	rows, err := sqlite.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var followedUser models.UserItem
		if err := rows.Scan(&followedUser.ID, &followedUser.Username, &followedUser.ProfileImg); err != nil {
			return nil, err
		}
		// if profile img is null, set it to a default image
		if followedUser.ProfileImg == "" {
			followedUser.ProfileImg = "profileImage.png"
		}
		following = append(following, followedUser)
	}

	return following, nil
}

// DeclineFollowRequest removes the follow relationship from the database
func DeclineFollowRequest(followerID int, followedID int) error {
	// Prepare the SQL statement
	query := `DELETE FROM followers WHERE follower_id = ? AND followed_id = ?`

	// Execute the query
	_, err := sqlite.DB.Exec(query, followerID, followedID)
	if err != nil {
		log.Println("Error deleting follow request:", err)
		return err
	}
	return nil
}

// AcceptFollowRequest updates the status of a follow request to accepted
func AcceptFollowRequest(followerID int, followedID int) error {
	// Prepare the SQL statement
	query := `UPDATE followers SET status = 'accepted' WHERE follower_id = ? AND followed_id = ? AND status = 'pending'`

	// Execute the query
	_, err := sqlite.DB.Exec(query, followerID, followedID)
	if err != nil {
		log.Println("Error updating follow request status:", err)
		return err
	}
	return nil
}
