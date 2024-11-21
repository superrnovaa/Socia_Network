package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"fmt"
	"log"
)

func GetUserByEmailOrUsername(emailOrUsername string) (models.User, error) {
	var user models.User
	err := sqlite.DB.QueryRow("SELECT id, username, email, first_name, last_name, nickname, date_of_birth, about_me, is_public, avatar_url, password FROM users WHERE email = $1 OR username = $1", emailOrUsername).Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.Nickname, &user.DateOfBirth, &user.AboutMe, &user.IsPublic, &user.AvatarURL, &user.Password)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func CreateUser(user models.User) (int, error) {
	// Set is_public to "public" by default if not provided
	if !user.IsPublic {
		user.IsPublic = true // Default to true if not provided
	}

	result, err := sqlite.DB.Exec(`
		INSERT INTO users (username, email, password, first_name, last_name, nickname, date_of_birth, about_me, avatar_url, is_public)
		VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, COALESCE(NULLIF(?, ''), 'ProfileImage.png'), ?)
	`, user.Username, user.Email, user.Password, user.FirstName, user.LastName, user.Nickname, user.DateOfBirth, user.AboutMe, user.AvatarURL, user.IsPublic)

	if err != nil {
		log.Printf("Error creating user: %v", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		return 0, err
	}

	return int(id), nil
}
func GetUserItemByID(userID int) (models.UserItem, error) {
	var user models.UserItem
	err := sqlite.DB.QueryRow("SELECT id, username, avatar_url FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.ProfileImg)
	if err != nil {
		return models.UserItem{}, err
	}
	return user, nil
}

func GetUserByUsername(username string) (models.User, error) {
	var user models.User
	row := sqlite.DB.QueryRow("SELECT id, username, email, first_name, last_name, nickname, date_of_birth, about_me, is_public, avatar_url FROM users WHERE username = ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.Nickname, &user.DateOfBirth, &user.AboutMe, &user.IsPublic, &user.AvatarURL)
	if err != nil {
		log.Printf("Error getting user by username: %v", err)
		return user, err
	}
	return user, nil
}

func GetUserByID(userID int) (*models.User, error) {
	var user models.User
	row := sqlite.DB.QueryRow("SELECT id, username, email FROM users WHERE id = ?", userID)
	err := row.Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		return nil, err
	}
	return &user, nil
}

func UpdateUser(user models.User) error {
	_, err := sqlite.DB.Exec(`
		UPDATE users 
		SET first_name = ?, last_name = ?, nickname = ?, date_of_birth = ?, 
			about_me = ?, avatar_url = COALESCE(NULLIF(?, ''), 'ProfileImage.png'), is_public = ? 
		WHERE id = ?`,
		user.FirstName, user.LastName, user.Nickname, user.DateOfBirth,
		user.AboutMe, user.AvatarURL, user.IsPublic, user.ID)
	if err != nil {
		log.Printf("Error updating user: %v", err)
		return err
	}
	return nil
}

func DeleteUser(userID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		return err
	}
	return nil
}

// FollowUnfollowUser handles both following and unfollowing a user
func FollowUnfollowUser(followerID, followedID int, action string) error {
	var query string
	switch action {
	case "follow":
		query = `INSERT OR IGNORE INTO followers (follower_id, followed_id, status) 
                 VALUES (?, ?, 'pending')`
	case "unfollow":
		query = `DELETE FROM followers 
                 WHERE follower_id = ? AND followed_id = ?`
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	_, err := sqlite.DB.Exec(query, followerID, followedID)
	if err != nil {
		log.Printf("Error %sing user: %v", action, err)
		return err
	}
	return nil
}

// GetFollowerCount returns the number of followers for a given user
func GetFollowerCount(userID int) (int, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM followers WHERE followed_id = ? AND status = 'accepted'", userID).Scan(&count)
	if err != nil {
		log.Printf("Error counting followers: %v", err)
		return 0, err
	}
	return count, nil
}

// GetAllUsers retrieves all users from the database
func GetAllUsersExcluding(Id int) ([]models.UserItem, error) {
	var users []models.UserItem
	rows, err := sqlite.DB.Query("SELECT id, username, avatar_url FROM users WHERE id != $1", Id) 
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.UserItem
		if err := rows.Scan(&user.ID, &user.Username, &user.ProfileImg); err != nil {
			return nil, err
		}
		// Check if ProfileImg is empty and set to "ProfileImage.png" if it is
		if user.ProfileImg == "" {
			user.ProfileImg = "ProfileImage.png"
		}
		users = append(users, user)
	}
	fmt.Println(users)
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Add this new function
func GetTopEngagedUsers(limit int) ([]models.UserItem, error) {
	query := `
		SELECT u.id, u.username, u.avatar_url, COUNT(p.id) as post_count
		FROM users u
		LEFT JOIN posts p ON u.id = p.user_id
		GROUP BY u.id
		ORDER BY post_count DESC
		LIMIT ?
	`
	rows, err := sqlite.DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserItem
	for rows.Next() {
		var user models.UserItem
		if err := rows.Scan(&user.ID, &user.Username, &user.ProfileImg, &user.PostCount); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func GetUserPostCount(userID int) (int, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		log.Printf("Error counting user posts: %v", err)
		return 0, err
	}
	return count, nil
}

func GetUserIdByUsername(username string) (int, error) {
	var userId int
	row := sqlite.DB.QueryRow("SELECT id FROM users WHERE username = ?", username)
	err := row.Scan(&userId)
	if err != nil {
		log.Printf("Error getting user by username: %v", err)
		return 0, err
	}
	return userId, nil
}
