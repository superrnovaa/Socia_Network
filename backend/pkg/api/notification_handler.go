package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"encoding/json"
	"net/http"
)

// GetNotificationsHandler handles the request to fetch notifications for a user
func GetAllNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch notifications from the database
	notifications, err := query.GetNotificationsQuery(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode notifications to JSON and send the response
	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		http.Error(w, "Failed to encode notifications", http.StatusInternalServerError)
	}
}

// GetNotificationsHandler handles the request to fetch notifications for a user
func GetNewNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch notifications from the database
	notifications, err := query.GetNewNotificationsQuery(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode notifications to JSON and send the response
	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		http.Error(w, "Failed to encode notifications", http.StatusInternalServerError)
	}
}

func MarkNotificationReadHandler(w http.ResponseWriter, r *http.Request) {

	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Implement logic to mark a notification as read
	err = query.MarkAllNotificationsReadQuery(user.ID)
	if err != nil {
		http.Error(w, "Failed to update notifications", http.StatusInternalServerError)
		return
	}
}

// NotificationCountUnreadHandler retrieves the count of unread notifications for the authenticated user.
func NotificationCountUnreadHandler(w http.ResponseWriter, r *http.Request) {
	// Get the current user's ID from the context
	currentUser, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch the count of unread notifications from the database
	count, err := query.CountUnreadNotificationsQuery(currentUser.ID)
	if err != nil {
		http.Error(w, "Failed to fetch unread notification count", http.StatusInternalServerError)
		return
	}

	// Respond with the count in JSON format
	response := map[string]int{"unread_count": count}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

