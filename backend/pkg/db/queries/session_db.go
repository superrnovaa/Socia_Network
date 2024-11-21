package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"log"
	"time"

	"github.com/gofrs/uuid"
)

// CreateSession creates a new session for a user and returns the session ID
func CreateSession(userID int) (string, error) {
	sessionID, err := uuid.NewV4()
	if err != nil {
		log.Printf("Error generating UUID: %v", err)
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	log.Printf("Creating session for user ID: %d", userID) // Add this line

	_, err = sqlite.DB.Exec(`
		INSERT OR REPLACE INTO sessions (id, user_id, expires_at) 
		VALUES (?, ?, ?)
	`, sessionID.String(), userID, expiresAt)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return "", err
	}

	return sessionID.String(), nil
}

// DeleteSession removes a session from the database
func DeleteSession(sessionID string) error {
	_, err := sqlite.DB.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
		return err
	}
	return nil
}

// GetUserIDBySession retrieves the user ID associated with a given session ID
func GetUserIDBySession(sessionID string) (int, error) {
	log.Println(sessionID)
	var userID int
	err := sqlite.DB.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expires_at > ?", sessionID, time.Now()).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Session not found or expired
		}
		log.Printf("Error getting user ID by session: %v", err)
		return 0, err
	}
	return userID, nil
}

// RefreshSession extends the expiration time of a session
func RefreshSession(sessionID string) error {
	newExpiresAt := time.Now().Add(24 * time.Hour)
	_, err := sqlite.DB.Exec("UPDATE sessions SET expires_at = ? WHERE id = ?", newExpiresAt, sessionID)
	if err != nil {
		log.Printf("Error refreshing session: %v", err)
		return err
	}
	return nil
}

// DeleteExpiredSessions removes all expired sessions from the database
func DeleteExpiredSessions() error {
	_, err := sqlite.DB.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now())
	if err != nil {
		log.Printf("Error deleting expired sessions: %v", err)
		return err
	}
	return nil
}

// IsSessionValid checks if a session is valid and not expired
func IsSessionValid(sessionID string) (bool, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? AND expires_at > ?", sessionID, time.Now()).Scan(&count)
	if err != nil {
		log.Printf("Error checking session validity: %v", err)
		return false, err
	}
	return count > 0, nil
}

// RemoveExistingSession removes any existing session for a user
func RemoveExistingSession(userID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Error removing existing session: %v", err)
		return err
	}
	return nil
}

// GetSessionUser retrieves the user associated with a given session ID
func GetSessionUser(sessionID string) (*models.User, error) {
	var user models.User
	var nickname, aboutMe, avatarURL sql.NullString

	err := sqlite.DB.QueryRow(`
		SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.nickname, u.date_of_birth, u.about_me, u.avatar_url
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > ?
	`, sessionID, time.Now()).Scan(
		&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName,
		&nickname, &user.DateOfBirth, &aboutMe, &avatarURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Session not found or expired
		}
		log.Printf("Error getting user by session: %v", err)
		return nil, err
	}

	// Handle NULL values
	user.Nickname = nickname.String
	user.AboutMe = aboutMe.String
	user.AvatarURL = avatarURL.String

	return &user, nil
}
