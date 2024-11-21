package post

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/websocket"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func isImageFile(file multipart.File) bool {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return false
	}
	contentType := http.DetectContentType(buffer)
	return strings.HasPrefix(contentType, "image/")
}

func AddCommentHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		content := r.FormValue("content")
		postID := r.FormValue("postId")
		if postID == "" {
			http.Error(w, "Missing post ID", http.StatusBadRequest)
			return
		}

		postIDInt, err := strconv.Atoi(postID)
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		comment := models.Comment{
			PostID: postIDInt,
			User: models.SafeUser{
				ID:        user.ID,
				Username:  user.Username,
				AvatarURL: user.AvatarURL,
			},
			Content: content,
		}

		// Handle file upload
		file, handler, err := r.FormFile("file")
		if err != nil {
			comment.File = "" // Set to empty string if no file is uploaded
		} else {
			defer file.Close()

			if !isImageFile(file) {
				http.Error(w, "Only image files are allowed", http.StatusBadRequest)
				return
			}

			// Reset file pointer after reading
			file.Seek(0, 0)

			// Generate a unique filename
			fileExt := filepath.Ext(handler.Filename)
			filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)

			// Create the uploads directory if it doesn't exist
			uploadsDir := "../../pkg/db/uploads"
			if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
				http.Error(w, "Failed to create uploads directory", http.StatusInternalServerError)
				return
			}

			// Create a new file in the uploads directory
			filePath := filepath.Join(uploadsDir, filename)
			dst, err := os.Create(filePath)
			if err != nil {
				http.Error(w, "Failed to create file", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			// Copy the uploaded file to the destination file
			if _, err := io.Copy(dst, file); err != nil {
				http.Error(w, "Failed to copy file data", http.StatusInternalServerError)
				return
			}

			// Store the new filename in the comment
			comment.File = filename
		}

		newComment, err := query.AddCommentQuery(comment)
		if err != nil {
			http.Error(w, "Failed to add comment", http.StatusInternalServerError)
			return
		}

		notifyedID, err := query.GetUserIDFromPostID(postIDInt)
		if err != nil {
			http.Error(w, "Failed to get user id from post", http.StatusInternalServerError)
			return
		}

		// Create a notification for the followee
		notification := models.Notification{
			NotifiedUserID:  notifyedID,
			NotifyingUserId: user.ID,
			ObjectID:        comment.PostID,
			//Object:           string(comment.PostID),
			Type:            "comment",
			Content:         user.Username + " Commented on your Post.",
			IsRead:          false,
			CreatedAt:       time.Now(),
			NotifyingImage:  user.AvatarURL, 
		}

		//  check if notifier is the notified user
		if notifyedID != user.ID {
			// Insert the notification into the database
		    NId, err := query.CreateNotification(notification); 
			if err != nil {
				http.Error(w, "Failed to create notification", http.StatusInternalServerError)
				return
			}
			notification.ID = int(NId)

			// Send the notification to the user via WebSocket
			websocket.SendNotificationToUser(appCore.Hub, notifyedID, notification)

		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(newComment); err != nil {
			http.Error(w, "Error encoding new comment to JSON", http.StatusInternalServerError)
			return
		}
	}
}

func GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("postId")
	if postID == "" {
		http.Error(w, "Missing post ID", http.StatusBadRequest)
		return
	}

	comments, err := query.GetCommentsQuery(postID)
	if err != nil {
		http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
		return
	}

	for i, comment := range comments {
		reactions, err := query.GetReactionsByContent(nil, &comment.ID)
		if err != nil {
			http.Error(w, "Failed to get reactions", http.StatusInternalServerError)
			return
		}
		comments[i].Reactions = reactions

		// Get user's reaction if user is authenticated
		if userID, ok := r.Context().Value("user_id").(int); ok {
			userReaction, err := query.GetUserReaction(userID, nil, &comment.ID)
			if err != nil {
				http.Error(w, "Failed to get user reaction", http.StatusInternalServerError)
				return
			}
			comments[i].UserReaction = userReaction
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, "Error encoding comments to JSON", http.StatusInternalServerError)
		return
	}
}

func UpdateCommentHandler(w http.ResponseWriter, r *http.Request) {
	var comment models.Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = query.UpdateCommentQuery(comment)
	if err != nil {
		http.Error(w, "Failed to update comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}


