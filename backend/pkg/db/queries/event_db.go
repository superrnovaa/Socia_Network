package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"fmt"
	"log"
)

func CreateEventQuery(newEvent models.Event) (int64, error) {
	result, err := sqlite.DB.Exec("INSERT INTO events (group_id, creator_id, title, description, event_date) VALUES (?, ?, ?, ?, ?)",
		newEvent.GroupID, newEvent.Creator.ID, newEvent.Title, newEvent.Description, newEvent.EventDate)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		return 0, err
	}

	// Get the last inserted ID
	eventID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return 0, err
	}

	return eventID, nil // Return the event ID
}

func GetEventsQuery(groupID int, userID int) ([]models.Event, error) {
	var events []models.Event
	query := `
		SELECT e.id, e.title, e.description, e.event_date, e.created_at, 
			   e.group_id, u.id AS creator_id, u.username AS creator_username, u.avatar_url AS creator_avatar
		FROM events e
		JOIN users u ON e.creator_id = u.id
		WHERE e.group_id = ?
		ORDER BY e.created_at DESC` // Sort by event_date from newest to oldest

	rows, err := sqlite.DB.Query(query, groupID)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event models.Event
		var creatorID int
		var creatorUsername string
		var creatorAvatar string // Variable to hold the avatar URL

		if err := rows.Scan(&event.ID, &event.Title, &event.Description, &event.EventDate, &event.CreatedAt, &event.GroupID, &creatorID, &creatorUsername, &creatorAvatar); err != nil {
			fmt.Println(err)
			return nil, err
		}

		 // Get the user's response for the event
		 userResponse, err := GetEventRespondQuery(event.ID, userID)
		 if err != nil {
			return nil, err
		 }
		 event.UserResponse = userResponse

		// Populate the Creator field
		event.Creator = models.SafeUser{ID: creatorID, Username: creatorUsername, AvatarURL: creatorAvatar} // Include avatar

		events = append(events, event)
	}

	return events, nil
}

func GetSingleEventQuery(eventID int) (models.Event, error) {
	var e models.Event
	err := sqlite.DB.QueryRow("SELECT id, group_id, creator_id, title, description, event_date, created_at FROM events WHERE id = ?", eventID).
		Scan(&e.ID, &e.GroupID, &e.Creator, &e.Title, &e.Description, &e.EventDate, &e.CreatedAt)
	if err != nil {
		log.Printf("Error retrieving single event: %v", err)
		return e, err
	}
	return e, nil
}

func UpdateEventQuery(updatedEvent models.Event) error {
	_, err := sqlite.DB.Exec("UPDATE events SET title = ?, description = ?, event_date = ? WHERE id = ?",
		updatedEvent.Title, updatedEvent.Description, updatedEvent.EventDate, updatedEvent.ID)
	if err != nil {
		log.Printf("Error updating event: %v", err)
		return err
	}
	return nil
}

func DeleteEventQuery(eventID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM events WHERE id = ?", eventID)
	if err != nil {
		log.Printf("Error deleting event: %v", err)
		return err
	}
	return nil
}

func RespondToEventQuery(response models.EventResponse) error {
	_, err := sqlite.DB.Exec("INSERT OR REPLACE INTO event_responses (event_id, user_id, response) VALUES (?, ?, ?)",
		response.EventID, response.User.ID, response.Response)
	if err != nil {
		log.Printf("Error responding to event: %v", err)
		return err
	}
	return nil
}

func GetEventResponsesQuery(eventID int) ([]models.EventResponse, error) { 
	var responses []models.EventResponse

	query := `
		SELECT r.id, r.event_id, r.user_id, u.username, u.avatar_url, r.response
		FROM event_responses r
		JOIN users u ON r.user_id = u.id
		WHERE r.event_id = $1
	` 

	rows, err := sqlite.DB.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var response models.EventResponse
		if err := rows.Scan(&response.ID, &response.EventID, &response.User.ID, &response.User.Username, &response.User.AvatarURL, &response.Response); err != nil { // {{ edit_3 }}
			return nil, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func GetEventRespondQuery(eventID int, userID int) (string, error) {
    var response string
    query := `
        SELECT response
        FROM event_responses
        WHERE event_id = $1 AND user_id = $2
    `

    err := sqlite.DB.QueryRow(query, eventID, userID).Scan(&response)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", nil // No response found
        }
        return "", err // Return the error if something went wrong
    }
    return response, nil
}
