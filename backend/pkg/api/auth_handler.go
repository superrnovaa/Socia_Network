package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/models"
	"backend/pkg/utilities"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var credentials struct {
		EmailOrUsername string `json:"emailOrUsername"`
		Password        string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := query.GetUserByEmailOrUsername(credentials.EmailOrUsername)
	if err != nil {
		sendErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	user.Following, user.Followers, err = query.GetUserStats(user.ID)
	if err != nil {
		http.Error(w, "Error checking following and followers", http.StatusInternalServerError)
		return
	}

	user.Notifications, err = query.CountUnreadNotificationsQuery(user.ID)
	if err != nil {
		http.Error(w, "Error checking notifications", http.StatusInternalServerError)
		return
	}

	// Remove any existing session for the user
	if err := query.RemoveExistingSession(user.ID); err != nil {
		http.Error(w, "Could not remove existing session", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		sendErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	sessionID, err := query.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
		Domain:   "localhost",
	})

	// Send a JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user.SafeUser(),
	})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := models.User{
		Username:    r.FormValue("username"),
		Email:       r.FormValue("email"),
		Password:    r.FormValue("password"),
		FirstName:   r.FormValue("firstName"),
		LastName:    r.FormValue("lastName"),
		Nickname:    r.FormValue("nickname"),
		DateOfBirth: r.FormValue("dateOfBirth"),
		AboutMe:     r.FormValue("aboutMe"),
	}

	// Handle file upload
	file, header, err := r.FormFile("profileImg")
	if err == nil {
		defer file.Close()
		filename, err := utilities.SaveFile(file, header)
		if err != nil {
			sendErrorResponse(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		user.AvatarURL = filename
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Could not hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	err = utilities.ValidateUser(user)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !isValidUsername(user.Username) {
		http.Error(w, "Invalid username. Use only letters, numbers, underscores, and hyphens.", http.StatusBadRequest)
		return
	}

	// Create the user and get the user ID
	userID, err := query.CreateUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a session for the new user
	sessionID, err := query.CreateSession(userID)
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}

	// Set the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
	})

	// Send a JSON response with user data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Signup successful",
		"user":    user.SafeUser(),
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "No session found", http.StatusBadRequest)
		return
	}

	if err := query.DeleteSession(cookie.Value); err != nil {
		http.Error(w, "Could not delete session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func sendSuccessResponse(w http.ResponseWriter, message int, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]int{"message": message})
}

func CheckSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		if err == http.ErrNoCookie {
			// No cookie found, user is not logged in
			sendJSONResponse(w, map[string]interface{}{
				"isLoggedIn": false,
				"user":       nil,
			})
			return
		}
		// Log the error for debugging
		log.Printf("Error retrieving cookie: %v", err)
		http.Error(w, "Error checking session", http.StatusInternalServerError)
		return
	}

	user, err := query.GetSessionUser(cookie.Value)
	if err != nil {
		// Log the error for debugging
		log.Printf("Error retrieving user from session: %v", err)
		http.Error(w, "Error checking session", http.StatusInternalServerError)
		return
	}

	if user == nil {
		sendJSONResponse(w, map[string]interface{}{
			"isLoggedIn": false,
			"user":       nil,
		})
		return
	}

	user.Following, user.Followers, err = query.GetUserStats(user.ID)
	if err != nil {
		// Log the error for debugging
		log.Printf("Error checking following and followers: %v", err)
		http.Error(w, "Error checking following and followers", http.StatusInternalServerError)
		return
	}

	user.Notifications, err = query.CountUnreadNotificationsQuery(user.ID)
	if err != nil {
		// Log the error for debugging
		log.Printf("Error checking notifications: %v", err)
		http.Error(w, "Error checking notifications", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{
		"isLoggedIn": true,
		"user":       user.SafeUser(),
	})
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func isValidUsername(username string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username)
}
