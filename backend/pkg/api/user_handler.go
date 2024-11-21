package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"backend/pkg/utilities"

	"golang.org/x/crypto/bcrypt"
)

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		fmt.Println(err)
	}
	// Retrieve users from the database, excluding the specified UserId
	users, err := query.GetAllUsersExcluding(user.ID) // Implement this function in your db package
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode users to JSON and send the response
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the username from the URL path
	path := r.URL.Path
	parts := strings.Split(path, "/") // Split the path by "/"
	if len(parts) < 4 {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	username := parts[3]
	user, err := query.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found or has been deleted"})
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user.Following, user.Followers, err = query.GetUserStats(user.ID)
	if err != nil {
		http.Error(w, "Internal server Error", http.StatusInternalServerError)
		return
	}
	user.Notifications, err = query.CountUnreadNotificationsQuery(user.ID)

	if err != nil {
		http.Error(w, "Internal server Error", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, "Error fetching user stats", http.StatusInternalServerError)
		return
	}

	// Get post count
	postCount, err := query.GetUserPostCount(user.ID)
	if err != nil {
		http.Error(w, "Error fetching post count", http.StatusInternalServerError)
		return
	}
	user.PostCount = postCount

	requestUser, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		// If there's no authenticated user, we'll treat them as a public viewer
		requestUser = &models.User{ID: 0}
	}

	FollowState, err := query.CheckIfUserFollows(requestUser.ID, user.ID)
	if err != nil {
		http.Error(w, "Error checking follow status", http.StatusInternalServerError)
		return
	}
	user.FollowState = FollowState

	// Check if the authenticated user is viewing their own profile
	isOwner := user.ID == requestUser.ID

	// Add the isOwner flag to the user response
	response := map[string]interface{}{
		"user":    user,
		"isOwner": isOwner,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// GetImageHandler serves the image file
func GetImageHandler(w http.ResponseWriter, r *http.Request) {
	imageName := r.URL.Query().Get("imageName") // Get the image name from the query parameter

	imagePath := filepath.Join("../../pkg/db/uploads", imageName) 

	// Check if the file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Serve the image file
	http.ServeFile(w, r, imagePath)
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var updatedUser models.User
	if err := r.ParseMultipartForm(10 << 20); err != nil { // Parse multipart form
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Decode user fields from form
	updatedUser.FirstName = r.FormValue("firstName")
	updatedUser.LastName = r.FormValue("lastName")
	updatedUser.Nickname = r.FormValue("nickname")
	updatedUser.DateOfBirth = r.FormValue("dateOfBirth")
	updatedUser.AboutMe = r.FormValue("aboutMe")
	updatedUser.Password = r.FormValue("password")
	updatedUser.IsPublic = r.FormValue("isPublic") == "true"
	updatedUser.ID = user.ID // Ensure the user ID is not changed

	isAvatarDeleted := r.FormValue("isAvatarDeleted") == "true"

	// Handle file upload or deletion
	if file, header, err := r.FormFile("profileImg"); err == nil {
		defer file.Close()

		filename, err := utilities.SaveFile(file, header)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		updatedUser.AvatarURL = filename
	} else if isAvatarDeleted {
		// Delete the existing avatar file if it exists
		if user.AvatarURL != "" && user.AvatarURL != "ProfileImage.png" {
			err := os.Remove(filepath.Join("../../pkg/db/uploads", user.AvatarURL))
			if err != nil {
				log.Printf("Error deleting avatar file: %v", err)
			}
		}
		updatedUser.AvatarURL = "" // Set to empty string or default value
	} else {
		updatedUser.AvatarURL = user.AvatarURL // Keep the existing avatar URL
	}

	// Hash password if it's provided
	if updatedUser.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedUser.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Could not hash password", http.StatusInternalServerError)
			return
		}
		updatedUser.Password = string(hashedPassword)
	} else {
		updatedUser.Password = user.Password // Retain the old password if not updated
	}

	updatedUser.IsPublic = r.FormValue("isPublic") == "true"

	// Update the user in the database
	err = query.UpdateUser(updatedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send a JSON response with updated user data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    updatedUser.SafeUser(),
	})
}


func GetTopEngagedUsersHandler(w http.ResponseWriter, r *http.Request) {
	limit := 3 // You can make this configurable if needed
	users, err := query.GetTopEngagedUsers(limit)
	if err != nil {
		http.Error(w, "Failed to fetch top engaged users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
