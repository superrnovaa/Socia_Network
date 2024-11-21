package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"fmt"
	"log"
	"strings"
	"database/sql"
)

func CreateNotification(notification models.Notification) (int64, error) {
	query := `
		INSERT INTO notifications (notifiedUser_id, notifyingUser_id, type, object, object_id, content, is_read, created_at)
		VALUES (?, ?, ?, NULLIF(?, 0), NULLIF(?, ''), ?, ?, ?)
	`
	result, err := sqlite.DB.Exec(query, notification.NotifiedUserID, notification.NotifyingUserId, notification.Type, notification.Object, notification.ObjectID, notification.Content, notification.IsRead, notification.CreatedAt)
	if err != nil {
		log.Printf("Error inserting notification: %v", err)
		return 0, err
	}

	// Get the ID of the newly inserted notification
	notificationID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return 0, err
	}

	return notificationID, nil // Return the ID of the newly created notification
}

func GetNotificationsQuery(userID int) ([]models.Notification, error) {
	rows, err := sqlite.DB.Query("SELECT id, notifiedUser_id, notifyingUser_id, type, content, is_read, object, object_id, created_at FROM notifications WHERE notifiedUser_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		log.Printf("Error retrieving notifications: %v", err)
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.NotifiedUserID, &n.NotifyingUserId, &n.Type, &n.Content, &n.IsRead, &n.Object, &n.ObjectID, &n.CreatedAt); err != nil {
			log.Printf("Error scanning notification row: %v", err)
			return nil, err
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func GetNewNotificationsQuery(userID int) ([]models.Notification, error) {
	rows, err := sqlite.DB.Query("SELECT id, notifiedUser_id, notifyingUser_id, type, content, is_read, object, object_id, created_at FROM notifications WHERE notifiedUser_id = ? AND is_read = 0 ORDER BY created_at DESC", userID)
	if err != nil {
		log.Printf("Error retrieving notifications: %v", err)
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.NotifiedUserID, &n.NotifyingUserId, &n.Type, &n.Content, &n.IsRead, &n.Object, &n.ObjectID, &n.CreatedAt); err != nil {
			log.Printf("Error scanning notification row: %v", err)
			return nil, err
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func ChangeNotificationType(NotifyingUserID int, NotifiedUserID int, Types []string, NewType string) error {
	// Prepare the SQL query with placeholders for the types
	query := "UPDATE notifications SET type = ? WHERE notifiedUser_id = ? AND notifyingUser_id = ? AND type IN ("

	// Create placeholders for the types
	placeholders := make([]string, len(Types))
	for i := range Types {
		placeholders[i] = "?"
	}

	// Join the placeholders with commas
	query += strings.Join(placeholders, ", ") + ")"

	// Create a slice of arguments for the query
	args := []interface{}{NewType, NotifiedUserID, NotifyingUserID}
	for _, t := range Types {
		args = append(args, t)
	}

	// Execute the query
	_, err := sqlite.DB.Exec(query, args...)
	if err != nil {
		log.Printf("Error changing notification type: %v", err)
		return err
	}
	return nil
}

func MarkNotificationReadQuery(notificationID int) error {
	_, err := sqlite.DB.Exec("UPDATE notifications SET is_read = 1 WHERE id = ?", notificationID)
	if err != nil {
		log.Printf("Error marking notification as read: %v", err)
		return err
	}
	return nil
}

func MarkAllNotificationsReadQuery(userID int) error {
	_, err := sqlite.DB.Exec("UPDATE notifications SET is_read = 1 WHERE notifiedUser_id = ?", userID)
	if err != nil {
		log.Printf("Error marking all notifications as read for user %d: %v", userID, err)
		return err
	}
	return nil
}

func CountUnreadNotificationsQuery(userID int) (int, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM notifications WHERE notifiedUser_id = ? AND is_read = 0", userID).Scan(&count)
	if err != nil {
		log.Printf("Error counting unread notifications: %v", err)
		return 0, err
	}
	return count, nil
}

func DeleteNotificationQuery(notificationID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM notifications WHERE id = ?", notificationID)
	if err != nil {
		log.Printf("Error deleting notification: %v", err)
		return err
	}
	return nil
}

func DeleteSpecificNotificationQuery(notifyingUserID int, notifiedUserID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM notifications WHERE notifyingUser_id = ? AND notifiedUser_id = ? AND type IN ('follow', 'follow_request')", notifyingUserID, notifiedUserID)
	if err != nil {
		log.Printf("Error deleting notification: %v", err)
		return err
	}
	return nil
}

func DeleteInviteNotificationQuery(notifyingUserID int, notifiedUserID int) (int64, error) {
	// First, retrieve the notification ID
	var notificationID int64
	err := sqlite.DB.QueryRow("SELECT id FROM notifications WHERE notifyingUser_id = ? AND notifiedUser_id = ? AND type = 'group_invitation'", notifyingUserID, notifiedUserID).Scan(&notificationID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No notification found to delete for notifyingUserID: %d and notifiedUserID: %d", notifyingUserID, notifiedUserID)
			return 0, nil // No notification found, return 0
		}
		log.Printf("Error retrieving notification ID: %v", err)
		return 0, err // Return error if there's an issue
	}

	// Now delete the notification
	_, err = sqlite.DB.Exec("DELETE FROM notifications WHERE notifyingUser_id = ? AND notifiedUser_id = ? AND type = 'group_invitation'", notifyingUserID, notifiedUserID)
	if err != nil {
		log.Printf("Error deleting notification: %v", err)
		return 0, err // Return error if deletion fails
	}

	return notificationID, nil // Return the deleted notification ID
}

// GetNotificationByDetails retrieves a notification based on notifyingUserID, notifiedUserID, objectID, types, and object.
func GetNotificationByDetails(notifyingUserID int, notifiedUserID int, objectID int, notificationTypes []string, object string) (models.Notification, error) {
	var notification models.Notification

	// Prepare the SQL query to retrieve the notification
	// Create a placeholder string for the IN clause
	placeholders := make([]string, len(notificationTypes))
	for i := range notificationTypes {
		placeholders[i] = "?" // Use "?" for placeholders
	}
	placeholderString := strings.Join(placeholders, ", ")

	query := fmt.Sprintf(`
		SELECT id, notifiedUser_id, notifyingUser_id, type, object, object_id, content, is_read, created_at
		FROM notifications
		WHERE notifyingUser_id = ? AND notifiedUser_id = ? AND type IN (%s) AND object_id = ? AND object = ?
		LIMIT 1
	`, placeholderString)

	// Create a slice of interface{} for the query arguments
	args := []interface{}{notifyingUserID, notifiedUserID}
	// Convert notificationTypes from []string to []interface{}
	for _, t := range notificationTypes {
		args = append(args, t)
	}
	// Append objectID and object as the last arguments
	args = append(args, objectID, object)

	// Execute the query
	err := sqlite.DB.QueryRow(query, args...).Scan(
		&notification.ID,
		&notification.NotifiedUserID,
		&notification.NotifyingUserId,
		&notification.Type,
		&notification.Object,
		&notification.ObjectID,
		&notification.Content,
		&notification.IsRead,
		&notification.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No notification found for notifyingUserID: %d, notifiedUserID: %d, types: %v, objectID: %d, object: %s", notifyingUserID, notifiedUserID, notificationTypes, objectID, object)
			return notification, nil // Return an empty notification and nil error
		}
		log.Printf("Error retrieving notification: %v", err)
		return notification, err // Return the error
	}

	return notification, nil // Return the found notification
}


// GetNotificationsByCommentID retrieves notifications of type "comment" for a specific comment ID
func GetNotificationsByCommentID(commentID int) ([]models.Notification, error) {
	query := `
		SELECT id, notifiedUser_id, type, object_id, is_read, created_at
		FROM notifications
		WHERE type = 'comment' AND object_id = ?` // Filter by type and object ID

	rows, err := sqlite.DB.Query(query, commentID)
	if err != nil {
		return nil, fmt.Errorf("error querying notifications: %v", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(&notification.ID, &notification.NotifiedUserID, &notification.Type, &notification.ObjectID, &notification.IsRead, &notification.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning notification: %v", err)
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over notifications: %v", err)
	}

	return notifications, nil
}

// GetNotificationsByDetails retrieves notifications based on notifyingUserID, objectID, and types.
func GetNotificationsByDetails(notifyingUserID int, objectID int, notificationTypes []string, object string) ([]models.Notification, error) {
	var notifications []models.Notification

	// Prepare the SQL query to retrieve the notifications
	// Create a placeholder string for the IN clause
	placeholders := make([]string, len(notificationTypes))
	for i := range notificationTypes {
		placeholders[i] = "?" // Use "?" for placeholders
	}
	placeholderString := strings.Join(placeholders, ", ")

	query := fmt.Sprintf(`
		SELECT id, notifiedUser_id, notifyingUser_id, type, object, object_id, content, is_read, created_at
		FROM notifications
		WHERE notifyingUser_id = ? AND object_id = ? AND type IN (%s)
	`, placeholderString)

	// Create a slice of interface{} for the query arguments
	args := []interface{}{notifyingUserID, objectID}
	// Convert notificationTypes from []string to []interface{}
	for _, t := range notificationTypes {
		args = append(args, t)
	}

	// Execute the query
	rows, err := sqlite.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error retrieving notifications: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(
			&notification.ID,
			&notification.NotifiedUserID,
			&notification.NotifyingUserId,
			&notification.Type,
			&notification.Object,
			&notification.ObjectID,
			&notification.Content,
			&notification.IsRead,
			&notification.CreatedAt,
		); err != nil {
			log.Printf("Error scanning notification row: %v", err)
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over notifications: %v", err)
		return nil, err
	}

	return notifications, nil // Return the array of notifications
}
