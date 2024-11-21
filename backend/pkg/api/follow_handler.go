package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/websocket"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"fmt"
	"time"
)

// FollowHandler handles the follow/unfollow requests
func InitFollowHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var followRequest models.FollowRequest

		// Decode the JSON request body
		if err := json.NewDecoder(r.Body).Decode(&followRequest); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		// get the user id from the token
		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		followRequest.FollowerID = user.ID
		if followRequest.FollowerID == followRequest.FollowedID {
			http.Error(w, "You cannot follow yourself", http.StatusBadRequest)
			return
		}
		followerUser, err := query.GetUserByID(followRequest.FollowerID)
		if err != nil {
			http.Error(w, "Failed to get follower details", http.StatusInternalServerError)
			return
		}
		// Create a notification for the followee
		notification := models.Notification{
			NotifiedUserID:  followRequest.FollowedID,
			NotifyingUserId: followRequest.FollowerID,
			Object:        followerUser.Username,
			ObjectID:       followerUser.ID,
			IsRead:          false,
			CreatedAt:       time.Now(),
			NotifyingImage:  followerUser.AvatarURL, 
		}

		if followRequest.ButtonState == "Following" {
			err := query.FollowUser(followRequest.FollowerID, followRequest.FollowedID, "accepted")
			if err != nil {
				http.Error(w, "Failed to accept follow request", http.StatusInternalServerError)
				return
			}
			
			notification.Content = followerUser.Username + " Started Following You."
			notification.Type = "follow"
			// Insert the notification into the database
			NId, err := query.CreateNotification(notification); 
			if err != nil {
				http.Error(w, "Failed to create notification", http.StatusInternalServerError)
				return
			}
			notification.ID = int(NId)
			// Send the notification to the followee via WebSocket
			websocket.SendNotificationToUser(appCore.Hub, followRequest.FollowedID, notification)

		} else if followRequest.ButtonState == "Follow" {
			err := query.UnfollowUser(followRequest.FollowerID, followRequest.FollowedID)
			if err != nil {
				http.Error(w, "Failed to unfollow user", http.StatusInternalServerError)
				return
			}
			// Retrieve the existing notification related to the unjoin action
			notification, err := query.GetNotificationByDetails(user.ID, followRequest.FollowedID, user.ID, []string{"follow", "follow_request"}, followerUser.Username)
			if err != nil {
				fmt.Printf("Error retrieving notification: %v", err)
				http.Error(w, "Failed to retrieve notification", http.StatusInternalServerError)
				return
			}
			if notification.ID != 0 {
			// Check if the notification is unread
			if !notification.IsRead {
				// Send the de-notification to the user via WebSocket
				websocket.SendDeNotificationToUser(appCore.Hub, followRequest.FollowedID) // Use the notification ID
			}
			fmt.Println("notification id",notification.ID)
			// Delete the existing notification
			err = query.DeleteNotificationQuery(notification.ID)
			if err != nil {
				http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
				return
			}
		}

		} else {
			err := query.FollowUser(followRequest.FollowerID, followRequest.FollowedID, "pending")
			if err != nil {
				http.Error(w, "Failed to send follow request", http.StatusInternalServerError)
				return
			}

			notification.Content = followerUser.Username + " sent you a follow request."
			notification.Type = "follow_request"
			// Insert the notification into the database
			NId, err := query.CreateNotification(notification); 
			if err != nil {
				http.Error(w, "Failed to create notification", http.StatusInternalServerError)
				return
			}
			notification.ID = int(NId)

			// Send the notification to the followee via WebSocket
			websocket.SendNotificationToUser(appCore.Hub, followRequest.FollowedID, notification)
		}

	}
}
func FollowingHandler(w http.ResponseWriter, r *http.Request) {

	// Extract the username from the URL path
	path := r.URL.Path
	parts := strings.Split(path, "/") // Split the path by "/"
	if len(parts) < 4 {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	username := parts[3]

	// Get user ID from username
	user, err := query.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	// Get following users
	following, err := query.GetFollowingDetails(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch followers data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(following)
}

func FollowersHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the username from the URL path
	path := r.URL.Path
	parts := strings.Split(path, "/") // Split the path by "/"
	if len(parts) < 4 {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	username := parts[3]

	// Get user ID from username
	user, err := query.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Get followers
	followers, err := query.GetFollowersDetails(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch followers data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(followers)
}

func FollowRequestHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID int    `json:"userId"`
		Action string `json:"action"`
	}

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Implement logic to handle the follow request
	switch request.Action {
	case "accept":
		// Logic to accept the follow request
		err := query.AcceptFollowRequest(request.UserID, user.ID)
		if err != nil {
			http.Error(w, "Failed to accept follow request", http.StatusInternalServerError)
			return
		}

		err = query.ChangeNotificationType(request.UserID, user.ID, []string{"follow_request"}, "follow")
		if err != nil {
			http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Follow request accepted")
	case "decline":
		// Logic to decline the follow request
		err := query.DeclineFollowRequest(request.UserID, user.ID)
		if err != nil {
			http.Error(w, "Failed to decline follow request", http.StatusInternalServerError)
			return
		}
		err = query.ChangeNotificationType(request.UserID, user.ID, []string{"follow_request"},"follow")
		if err != nil {
			http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Follow request declined")
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// Add this function to the existing file
func GetFollowersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	followers, err := query.GetFollowersDetails(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch followers data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(followers)
}
