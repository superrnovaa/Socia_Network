package post

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/utilities"
	"backend/pkg/websocket"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func CreatePostHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) // Limit to 10 MB
		if err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		title := r.FormValue("Title")
		content := r.FormValue("Content")
		privacy := r.FormValue("privacy")
		checkedUsersJSON := r.FormValue("checkedUserIds")

		if privacy != "public" && privacy != "private" && privacy != "almost_private" {
			http.Error(w, "Invalid privacy setting", http.StatusBadRequest)
			return
		}

		post := models.Post{
			Title:   title,
			Content: content,
			Privacy: privacy,
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			post.File = "" // Set to empty string if no file is uploaded
		} else {
			defer file.Close()
			newFileName, err := utilities.SaveFile(file, header)
			if err != nil {
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			post.File = newFileName // Set the new filename if a file is uploaded
		}

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		post.User = models.SafeUser{
			ID:        user.ID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		}

		postId, err := query.CreatePostQuery(post)
		if err != nil {
			http.Error(w, "Error creating post", http.StatusInternalServerError)
			return
		}

		if privacy == "almost_private" {
			var checkedUserIds []int
			if err := json.Unmarshal([]byte(checkedUsersJSON), &checkedUserIds); err != nil {
				http.Error(w, "Invalid user IDs", http.StatusBadRequest)
				return
			}

			for _, userId := range checkedUserIds {
				err := query.InsertPostViewer(postId, userId)
				if err != nil {
					http.Error(w, "Error inserting post viewers", http.StatusInternalServerError)
					return
				}
				// Create a notification for the followee
				notification := models.Notification{
					NotifiedUserID:  userId,
					NotifyingUserId: user.ID,
					ObjectID:        int(postId),
					Type:            "post",
					Content:         user.Username + " Added a New Post.",
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
				websocket.SendNotificationToUser(appCore.Hub, userId, notification)

			}

		} else {
			followers, err := query.GetFollowers(user.ID)
			if err != nil {
				http.Error(w, "Error getting followers", http.StatusInternalServerError)
				return
			}

			for _, follower := range followers {
				// Create a notification for the followee
				notification := models.Notification{
					NotifiedUserID:  follower,
					NotifyingUserId: user.ID,
					ObjectID:        int(postId),
					Type:            "post",
					Content:         user.Username + " Added a New Post.",
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
				websocket.SendNotificationToUser(appCore.Hub, follower, notification)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Post created successfully"))
	}
}

func GetPostsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	posts, err := query.GetPostsQuery(user.ID)
	if err != nil {
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}

	for i, post := range posts {
		reactions, err := query.GetReactionsByContent(&post.ID, nil)
		if err != nil {
			http.Error(w, "Failed to get reactions", http.StatusInternalServerError)
			return
		}
		posts[i].Reactions = reactions

		userReaction, err := query.GetUserReaction(user.ID, &post.ID, nil)
		if err != nil {
			http.Error(w, "Failed to get user reaction", http.StatusInternalServerError)
			return
		}
		posts[i].UserReaction = userReaction
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, "Error encoding posts to JSON", http.StatusInternalServerError)
	}
}

func UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	existingPost, err := query.GetSinglePostQuery(strconv.Itoa(postID), user.ID)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	if existingPost.User.ID != user.ID {
		http.Error(w, "You don't have permission to update this post", http.StatusForbidden)
		return
	}

	updatedPost := models.Post{
		ID:      postID,
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
		Privacy: r.FormValue("privacy"),
		User:    existingPost.User,
		File:    existingPost.File, // Keep existing file by default
	}

	// Check if the file should be cleared
	if r.FormValue("clearFile") == "true" {
		updatedPost.File = ""
	} else {
		// Handle new file upload if present
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			newFileName, err := utilities.SaveFile(file, header)
			if err != nil {
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			updatedPost.File = newFileName
		}
	}

	err = query.UpdatePostQuery(updatedPost)
	if err != nil {
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	if updatedPost.Privacy == "almost_private" {
		checkedUsersJSON := r.FormValue("checkedUserIds")
		var checkedUserIds []int
		if err := json.Unmarshal([]byte(checkedUsersJSON), &checkedUserIds); err != nil {
			http.Error(w, "Invalid user IDs", http.StatusBadRequest)
			return
		}

		err = query.UpdatePostViewers(postID, checkedUserIds)
		if err != nil {
			http.Error(w, "Error updating post viewers", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedPost)
}

func DeletePostHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		postID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		// Check if the user is the owner of the post
		existingPost, err := query.GetSinglePostQuery(strconv.Itoa(postID), user.ID)
		if err != nil {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		if existingPost.User.ID != user.ID {
			http.Error(w, "You don't have permission to delete this post", http.StatusForbidden)
			return
		}

		// Retrieve notifications related to the post
		notifications, err := query.GetNotificationsByDetails(user.ID, postID, []string{"post"}, "post")
		if err != nil {
			http.Error(w, "Failed to retrieve notifications", http.StatusInternalServerError)
			return
		}

		// Send de-notifications to users if notifications are unread
		for _, notification := range notifications {
			websocket.SendDeNotificationToUser(appCore.Hub, notification.NotifiedUserID)

			// Delete the existing notification
			err = query.DeleteNotificationQuery(notification.ID)
			if err != nil {
				http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
				return
			}
		}

		// Delete the post
		err = query.DeletePostQuery(postID)
		if err != nil {
			http.Error(w, "Failed to delete post", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Post deleted successfully"})
	}
}

// Add this function to the existing file
func GetSinglePostHandler(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("id")
	if postID == "" {
		http.Error(w, "Missing post ID", http.StatusBadRequest)
		return
	}

	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postIDInt, err := strconv.Atoi(postID)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	permitted, err := query.IsUserPermittedToViewPost(postIDInt, user.ID)
	if err != nil {
		log.Printf("Error checking post permissions: %v", err)
		http.Error(w, "Error checking post permissions", http.StatusInternalServerError)
		return
	}

	if !permitted {
		http.Error(w, "You do not have permission to view this post", http.StatusForbidden)
		return
	}

	post, err := query.GetSinglePostQuery(postID, user.ID)
	if err != nil {
		log.Printf("Error retrieving post: %v", err)
		http.Error(w, "Error retrieving post", http.StatusInternalServerError)
		return
	}

	reactions, err := query.GetReactionsByContent(&post.ID, nil)
	if err != nil {
		http.Error(w, "Failed to get reactions", http.StatusInternalServerError)
		return
	}
	post.Reactions = reactions

	// Get user's reaction if user is authenticated
	if userID, ok := r.Context().Value("user_id").(int); ok {
		userReaction, err := query.GetUserReaction(userID, &post.ID, nil)
		if err != nil {
			http.Error(w, "Failed to get user reaction", http.StatusInternalServerError)
			return
		}
		post.UserReaction = userReaction
	}

	comments, err := query.GetCommentsQuery(postID)
	if err != nil {
		log.Printf("Error retrieving comments: %v", err)
		http.Error(w, "Error retrieving comments", http.StatusInternalServerError)
		return
	}

	post.Comments = comments

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		log.Printf("Error encoding post to JSON: %v", err)
		http.Error(w, "Error encoding post to JSON", http.StatusInternalServerError)
	}
}

// Add this new function
func GetUserPostsHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	targetUser, err := query.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	currentUser, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		// If not authenticated, treat as public viewer
		currentUser = &models.User{ID: 0}
	}

	posts, err := query.GetUserPostsQuery(targetUser.ID, currentUser.ID)
	if err != nil {
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}

	for i, post := range posts {
		reactions, err := query.GetReactionsByContent(&post.ID, nil)
		if err != nil {
			http.Error(w, "Failed to get reactions", http.StatusInternalServerError)
			return
		}
		posts[i].Reactions = reactions

		userReaction, err := query.GetUserReaction(currentUser.ID, &post.ID, nil)
		if err != nil {
			http.Error(w, "Failed to get user reaction", http.StatusInternalServerError)
			return
		}
		posts[i].UserReaction = userReaction
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, "Error encoding posts to JSON", http.StatusInternalServerError)
	}
}

// Add or update this handler
func CreateGroupPostHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) // Limit to 10 MB
		if err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		title := r.FormValue("Title")
		content := r.FormValue("Content")
		privacy := r.FormValue("privacy")
		groupname := r.FormValue("groupname")

		groupname, err = url.QueryUnescape(r.FormValue("groupname"))
		if err != nil {
			http.Error(w, "Invalid group name", http.StatusBadRequest)
			return
		}

		groupID, err := query.GetGroupIDByName(groupname)
		if err != nil {
			http.Error(w, "Invalid group name", http.StatusBadRequest)
			return
		}

		post := models.Post{
			Title:   title,
			Content: content,
			Privacy: privacy,
			Group: &models.Group{
				ID:   groupID,
				Name: groupname,
			},
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			post.File = "" // Set to empty string if no file is uploaded
		} else {
			defer file.Close()
			newFileName, err := utilities.SaveFile(file, header)
			if err != nil {
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			post.File = newFileName // Set the new filename if a file is uploaded
		}

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		post.User = models.SafeUser{
			ID:        user.ID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		}

		postId, err := query.CreateGroupPostQuery(post)
		if err != nil {
			http.Error(w, "Error creating group post", http.StatusInternalServerError)
			return
		}
		groupMembers, err := query.GetGroupMembersAsUsers(groupID)
		if err != nil {
			http.Error(w, "Error finding group members", http.StatusInternalServerError)
		}

		for _, member := range groupMembers {
			// Create a notification for the followee
			notification := models.Notification{
				NotifiedUserID:  member.ID,
				NotifyingUserId: user.ID,
				ObjectID:        int(postId),
				Type:            "post",
				Content:         user.Username + " Added a New Post.",
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"postId": postId})
	}
}

// Add or update this handler
func GetGroupPostsHandler(w http.ResponseWriter, r *http.Request) {
	groupname := r.URL.Query().Get("groupname")
	if groupname == "" {
		http.Error(w, "Missing group name", http.StatusBadRequest)
		return
	}

	groupID, err := query.GetGroupIDByName(groupname)
	if err != nil {
		http.Error(w, "Invalid group name", http.StatusBadRequest)
		return
	}

	posts, err := query.GetGroupPostsQuery(groupID)
	if err != nil {
		http.Error(w, "Error retrieving group posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, "Error encoding posts to JSON", http.StatusInternalServerError)
	}
}
