package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/websocket"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func ReactHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reaction models.Reaction
		err := json.NewDecoder(r.Body).Decode(&reaction)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate that either post_id or comment_id is provided, but not both
		if (reaction.PostID == nil && reaction.CommentID == nil) || (reaction.PostID != nil && reaction.CommentID != nil) {
			http.Error(w, "Either post_id or comment_id must be provided, but not both", http.StatusBadRequest)
			return
		}

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		reaction.UserID = user.ID


		// Add, update, or remove the reaction
		exists, err := query.AddOrUpdateReaction(reaction)
		if err != nil {
			http.Error(w, "Failed to process reaction", http.StatusInternalServerError)
			return
		}
		userID, err := query.GetUserIDFromPostID(*reaction.PostID)
		if err != nil {
			http.Error(w, "Failed to get user id from post", http.StatusInternalServerError)
			return
		}
		if userID != user.ID {
		notifyedID, err := query.GetUserIDFromPostID(*reaction.PostID)
		if err != nil {
			http.Error(w, "Failed to get user id from post", http.StatusInternalServerError)
			return
		}
		if !exists {
			// Create a notification for the followee
			notification := models.Notification{
				NotifiedUserID:  notifyedID,
				NotifyingUserId: user.ID,
				ObjectID:        *reaction.PostID,
				Type:            "reaction",
				Content:         user.Username + " Reacted on your Post.",
				IsRead:          false,
				CreatedAt:       time.Now(),
				NotifyingImage:  user.AvatarURL,
			}

			// Check for existing notification
			existingNotification, err := query.GetNotificationByDetails(user.ID, notifyedID, *reaction.PostID, []string{"reaction"}, "")
			if err == nil {
				// If an existing notification is found, delete it
				err = query.DeleteNotificationQuery(existingNotification.ID)
				if err != nil {
					http.Error(w, "Failed to delete existing notification", http.StatusInternalServerError)
					return
				}
			}

			// Insert the new notification into the database
			NId, err := query.CreateNotification(notification)
			if err != nil {
				http.Error(w, "Failed to create notification", http.StatusInternalServerError)
				return
			}
			notification.ID = int(NId)

			// Send the notification to the user via WebSocket
			websocket.SendNotificationToUser(appCore.Hub, notifyedID, notification)
		} else {

			// Retrieve the notification based on notifyingUserID, notifiedUserID, and type
			notification, err := query.GetNotificationByDetails(user.ID, notifyedID, *reaction.PostID, []string{"reaction"}, "")
			if err != nil {
				fmt.Printf("Error retrieving notification: %v", err)
				http.Error(w, "Failed to retrieve notification", http.StatusInternalServerError)
				return // Handle the error appropriately
			}

			// Check if the notification is unread
			if !notification.IsRead {
				// Send the notification to the user via WebSocket
				websocket.SendDeNotificationToUser(appCore.Hub, notifyedID) // Use the notification ID
			}
			err = query.DeleteNotificationQuery(notification.ID)
			if err != nil {
				http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
				return
			}
		}
	}
		// Get updated reaction counts
		var reactionCounts map[string]int
		if reaction.PostID != nil {
			reactionCounts, err = query.GetReactionsByContent(reaction.PostID, nil)
		} else {
			reactionCounts, err = query.GetReactionsByContent(nil, reaction.CommentID)
		}
		if err != nil {
			http.Error(w, "Failed to get updated reaction counts", http.StatusInternalServerError)
			return
		}

		// Get the user's current reaction (if any)
		userReaction, err := query.GetUserReaction(user.ID, reaction.PostID, reaction.CommentID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Failed to get user reaction", http.StatusInternalServerError)
			return
		}

		// Prepare the response
		response := struct {
			Reactions    map[string]int   `json:"reactions"`
			UserReaction *models.Reaction `json:"user_reaction"`
		}{
			Reactions:    reactionCounts,
			UserReaction: userReaction,
		}

		// Return updated reaction counts and user reaction
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func GetAvailableReactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reactions, err := query.GetAvailableReactions()
	if err != nil {
		http.Error(w, "Failed to get available reactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reactions)
}
