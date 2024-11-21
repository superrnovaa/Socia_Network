package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/websocket"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	MAX_EVENT_TITLE_LENGTH       = 50
	MAX_EVENT_DESCRIPTION_LENGTH = 500
)

func CreateEventHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event models.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Validate title length
		if len(event.Title) > MAX_EVENT_TITLE_LENGTH {
			http.Error(w, fmt.Sprintf("Title must not exceed %d characters", MAX_EVENT_TITLE_LENGTH), http.StatusBadRequest)
			return
		}

		// Validate description length
		if len(event.Description) > MAX_EVENT_DESCRIPTION_LENGTH {
			http.Error(w, fmt.Sprintf("Description must not exceed %d characters", MAX_EVENT_DESCRIPTION_LENGTH), http.StatusBadRequest)
			return
		}

		// Ensure all required fields are filled
		if event.Title == "" || event.Description == "" || event.EventDate.IsZero() {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Get user ID from middleware
		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		event.Creator.ID = user.ID

		// Decode the URL-encoded group name
		decodedGroupName, err := url.QueryUnescape(event.GroupName)
		if err != nil {
			http.Error(w, "Invalid group name", http.StatusBadRequest)
			return
		}

		// Get group ID from decoded groupname
		groupID, err := query.GetGroupIDByName(decodedGroupName)
		if err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		event.GroupID = groupID // Set the group ID in the event
		event.CreatedAt = time.Now()

		eventId, err := query.CreateEventQuery(event)
		if err != nil {
			http.Error(w, "Failed to create event", http.StatusInternalServerError)
			return
		}
		groupMembers, err := query.GetGroupMembersAsUsers(groupID)
		if err != nil {
			http.Error(w, "Failed to get group members", http.StatusInternalServerError)
		}

		for _, member := range groupMembers {
			if member.ID != user.ID {
				// Create a notification for the followee
				notification := models.Notification{
					NotifiedUserID:  member.ID,
					NotifyingUserId: user.ID,
					ObjectID:        int(eventId),
					Object:          event.GroupName,
					Type:            "event_creation",
					Content:         user.Username + " from " + event.GroupName + " is inviting you to " + event.Title + ".",
					IsRead:          false,
					CreatedAt:       time.Now(),
					NotifyingImage:  user.AvatarURL,
				}

				// Insert the notification into the database
				NId, err := query.CreateNotification(notification)
				if err != nil {
					http.Error(w, "Failed to create notification", http.StatusInternalServerError)
					return
				}
				notification.ID = int(NId)
				// Send the notification to the user via WebSocket
				websocket.SendNotificationToUser(appCore.Hub, member.ID, notification)
			}
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("groupname") // Get groupname from query parameters

	// Get group ID from group name
	groupID, err := query.GetGroupIDByName(groupName)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get user ID from middleware
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch events using the group ID
	events, err := query.GetEventsQuery(groupID, user.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events) // Send events as JSON response
}


func RespondToEventHandler(w http.ResponseWriter, r *http.Request) {
	// Get the event ID from the query parameters
	eventID := r.URL.Query().Get("eventID")
	fmt.Println("worked")

	// Parse the request body
	var requestBody struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventIDInt, err := strconv.Atoi(eventID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Create an EventResponse instance
	eventResponse := models.EventResponse{
		EventID: eventIDInt,
		User: models.SafeUser{
			ID: user.ID,
		},
		Response: requestBody.Response,
	}

	// Call the query function to update the response in the database
	if err := query.RespondToEventQuery(eventResponse); err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to update response", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Response updated successfully"})
}

func GetEventResponsesHandler(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("eventID")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	responses, err := query.GetEventResponsesQuery(eventID)
	if err != nil {
		http.Error(w, "Failed to fetch responses", http.StatusInternalServerError) // {{ edit_7 }}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses) // Encode responses to JSON and send
}
