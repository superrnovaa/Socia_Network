package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware" // Import the middleware package for context functions
	"backend/pkg/models"
	"backend/pkg/utilities"
	"backend/pkg/websocket"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MAX_GROUP_TITLE_LENGTH       = 32  // Match frontend limit
	MAX_GROUP_DESCRIPTION_LENGTH = 200 // Match frontend limit
)

func CreateGroupHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Get title and description from form values
		title := r.FormValue("title")
		description := r.FormValue("description")

		// Validate title length
		if len(title) > MAX_GROUP_TITLE_LENGTH {
			http.Error(w, fmt.Sprintf("Title must not exceed %d characters", MAX_GROUP_TITLE_LENGTH), http.StatusBadRequest)
			return
		}

		// Validate description length
		if len(description) > MAX_GROUP_DESCRIPTION_LENGTH {
			http.Error(w, fmt.Sprintf("Description must not exceed %d characters", MAX_GROUP_DESCRIPTION_LENGTH), http.StatusBadRequest)
			return
		}

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		exists, err := query.CheckGroupNameExists(r.FormValue("title"))

		if err != nil {
			http.Error(w, "Error checking group name", http.StatusInternalServerError)
			return
		}

		if exists {
			w.WriteHeader(http.StatusConflict) // 409 Conflict
			http.Error(w, "Error checking group name", http.StatusConflict)
			return
		} else {
			w.WriteHeader(http.StatusOK) // 200 OK
			json.NewEncoder(w).Encode(map[string]bool{"exists": false})
		}

		newGroup := models.Group{
			Name:        r.FormValue("title"),
			Description: r.FormValue("description"),
			CreatorID:   user.ID,
		}

		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			filename, err := utilities.SaveFile(file, header)
			if err != nil {
				sendErrorResponse(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			newGroup.ImageURL = filename
		}

		groupID, err := query.CreateGroupQuery(newGroup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		membersStr := r.FormValue("members") // Get the members string

		var membersList []int
		if err := json.Unmarshal([]byte(membersStr), &membersList); err != nil {
			http.Error(w, "Invalid members format", http.StatusBadRequest)
			return
		}
		membersList = append(membersList, user.ID)

		for _, member := range membersList {
			if member == user.ID {
				err = query.AddGroupMember(groupID, member, "accepted", user.ID)
			} else {
				err = query.AddGroupMember(groupID, member, "pending", user.ID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Create a notification for the followee
				notification := models.Notification{
					NotifiedUserID:  member,
					NotifyingUserId: user.ID,
					Object:          newGroup.Name,
					ObjectID:        int(groupID),
					Type:            "group_invitation",
					Content:         user.Username + " invited you to '" + newGroup.Name + "' group",
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
				websocket.SendNotificationToUser(appCore.Hub, member, notification)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newGroup)
	}
}

func GetGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	groupname := r.URL.Query().Get("groupname")
	if groupname == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Try to fetch the group using the original name

	group, err := query.GetGroupByName(groupname)
	if err != nil {
		// If not found, try with URL-safe name
		urlSafeName := strings.ReplaceAll(groupname, " ", "-")
		group, err = query.GetGroupByName(urlSafeName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Group not found or has been deleted"})
			return
		}
	}

	// get user
	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// check if the user is a member of the group
	memberStatus, err := query.GetMemberStatus(currentUser.ID, group.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check if the user is the creator of the group
	isCreator, err := query.IsCreator(group.ID, currentUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// if status is pending, return a message
	if memberStatus == "pending" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "You have a pending invitation to this group"})
		return
	}

	// if status is accepted or creator, return the group
	if memberStatus == "accepted" || isCreator {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(group)
	}

	// if status is not accepted or creator, return a message
	if memberStatus != "accepted" && !isCreator {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "You are not a member of this group please send a request"})
	}

}

func GetGroupDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	groupname := r.URL.Query().Get("groupname")
	if groupname == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Try to fetch the group using the original name
	group, err := query.GetGroupByName(groupname)
	if err != nil {
		// If not found, try with URL-safe name
		urlSafeName := strings.ReplaceAll(groupname, " ", "-")
		group, err = query.GetGroupByName(urlSafeName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Group not found or has been deleted"})
			return
		}
	}

	// Get the current user's ID from the context
	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check the membership status of the current user in the group
	memberStatus, err := query.GetMemberStatus(currentUser.ID, group.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare the response with group details and membership status
	response := struct {
		Group        models.Group `json:"group"`
		MemberStatus string       `json:"member_status"`
	}{
		Group:        group,
		MemberStatus: memberStatus, // This could be "accepted", "pending", or "not a member"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetGroupsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	categorizedGroups, err := query.GetCategorizedGroups(currentUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categorizedGroups)
}

func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // Limit the size to 10 MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get the current user's ID from the context
	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupIDStr := r.FormValue("id")
	groupID, err := strconv.Atoi(groupIDStr) // Convert string to int
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	updatedGroup := models.Group{
		ID:          groupID,
		Name:        r.FormValue("title"),
		Description: r.FormValue("description"),
	}

	// Check if the user making the request is the creator of the group
	IsCreator, err := query.IsCreator(updatedGroup.ID, currentUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !IsCreator {
		http.Error(w, "Only the group creator can update the group", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := utilities.SaveFile(file, header)
		if err != nil {
			fmt.Println(err)
			sendErrorResponse(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		updatedGroup.ImageURL = filename
	}

	err = query.UpdateGroupQuery(updatedGroup)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve removed members from the form
	var removedMembers []int
	if memberData := r.MultipartForm.Value["removedMembers[]"]; len(memberData) > 0 {
		for _, memberIDStr := range memberData {
			memberID, err := strconv.Atoi(memberIDStr)
			if err != nil {
				http.Error(w, "Invalid member ID", http.StatusBadRequest)
				return
			}
			removedMembers = append(removedMembers, memberID)
		}
	}
	fmt.Println("updatedGroup:", updatedGroup)
	fmt.Println("remove members:", removedMembers)
	fmt.Println("image:", updatedGroup.ImageURL)

	// Call RemoveGroupMembers with the IDs to remove
	if len(removedMembers) > 0 {
		err = query.RemoveGroupMembers(groupID, removedMembers)
		if err != nil {
			http.Error(w, "Error removing group members", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(updatedGroup)
}

// DeleteGroupHandler deletes a group by ID
func DeleteGroupHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupname := r.URL.Query().Get("groupname") // Get groupname from query parameters

		// Retrieve group ID based on groupname
		groupID, err := query.GetGroupIDByName(groupname)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Get the current user's ID from the context
		currentUser, err := middleware.GetUserFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the user making the request is the creator of the group
		IsCreator, err := query.IsCreator(groupID, currentUser.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !IsCreator {
			http.Error(w, "Only the group creator can delete the group", http.StatusForbidden)
			return
		}

		err = query.DeleteGroupQuery(groupID)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		members, err := query.GetGroupMembersAsUsers(groupID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Send de-notifications to users if notifications are unread
		for _, member := range members {
			websocket.SendDeNotificationToUser(appCore.Hub, member.ID)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// InviteUsersHandler invites multiple users to a group
func InviteUsersHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		groupname := r.URL.Query().Get("groupname") // Get groupname from query parameters

		// Retrieve group ID based on groupname
		groupID, err := query.GetGroupIDByName(groupname)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Parse user IDs from the request body
		var requestBody struct {
			UserIDs []int `json:"userIDs"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userIDs := requestBody.UserIDs

		currentUser, err := middleware.GetUserFromContext(r) // Get the current user's ID
		if err != nil {
			fmt.Println(currentUser, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		for _, userID := range userIDs {
			isInGroup, err := query.IsUserInGroup(groupID, userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if isInGroup {
				http.Error(w, "User is already a member of the group", http.StatusConflict)
				return
			}
		}

		err = query.InviteUsersToGroup(groupID, userIDs, currentUser.ID) // Pass the inviter ID
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, member := range userIDs {
			// Create a notification for the followee
			notification := models.Notification{
				NotifiedUserID:  member,
				NotifyingUserId: currentUser.ID,
				Object:          groupname,
				ObjectID:        groupID,
				Type:            "group_invitation",
				Content:         currentUser.Username + " invited you to '" + groupname + "' group",
				IsRead:          false,
				CreatedAt:       time.Now(),
				NotifyingImage:  currentUser.AvatarURL,
			}

			// Insert the notification into the database
			NId, err := query.CreateNotification(notification)
			if err != nil {
				http.Error(w, "Failed to create notification", http.StatusInternalServerError)
				return
			}
			notification.ID = int(NId)
			// Send the notification to the user via WebSocket
			websocket.SendNotificationToUser(appCore.Hub, member, notification)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// AcceptInvitationHandler accepts a group invitation
func AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	groupID, err := strconv.Atoi(r.URL.Query().Get("group_id"))
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		fmt.Println(err)
	}

	err = query.AcceptGroupInvitation(groupID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = query.ChangeNotificationType(userID, user.ID, []string{"group_join_request"}, "group")
	if err != nil {
		http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RejectInvitationHandler rejects a group invitation
func RejectInvitationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	groupID, err := strconv.Atoi(r.URL.Query().Get("group_id"))
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		fmt.Println(err)
	}

	err = query.RejectGroupInvitation(groupID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = query.ChangeNotificationType(userID, user.ID, []string{"group_join_request"}, "group")
	if err != nil {
		http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMembersHandler removes multiple members from a group
func RemoveMembersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var request struct {
		GroupID int   `json:"group_id"`
		UserIDs []int `json:"user_ids"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the user making the request is the creator of the group
	creatorID, err := query.GetGroupCreatorID(request.GroupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the current user's ID from the context
	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if currentUser.ID != creatorID {
		http.Error(w, "Only the group creator can remove members", http.StatusForbidden)
		return
	}

	err = query.RemoveGroupMembers(request.GroupID, request.UserIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetGroupInvitationListHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("groupname") // Get groupname from query parameters

	// Get group ID from group name
	groupID, err := query.GetGroupIDByName(groupName)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	currentUser, err := middleware.GetUserFromContext(r)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all users excluding the current user
	users, err := query.GetAllUsersExcluding(currentUser.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	var invitations []models.InviteUser

	// Filter users who are not in the group
	for _, user := range users {
		status := "Invite" // Default status
		memberStatus, err := query.GetMemberStatus(user.ID, groupID)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error fetching user status", http.StatusInternalServerError)
			return
		}

		if memberStatus == "pending" {
			status = "Pending" // Set status to Pending if user has a pending status
		}

		invitation := models.InviteUser{
			User:   user,
			Status: status,
		}

		if memberStatus != "accepted" {
			invitations = append(invitations, invitation)
		}

	}

	// Return the invitation list as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

// CancelInvitationHandler cancels a group invitation for a user
func CancelInvitationHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		groupName := r.URL.Query().Get("groupname") // Get groupname from query parameters

		// Get group ID from group name
		groupID, err := query.GetGroupIDByName(groupName)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Define a struct to hold the incoming request body
		type requestBody struct {
			UserID int `json:"userID"` // Field to hold userID
		}

		var reqBody requestBody

		err = json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		currentUser, err := middleware.GetUserFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Println(groupID, reqBody.UserID, currentUser.ID)
		// Pass the inviter ID (current user's ID) to the CancelGroupInvitation function
		err = query.CancelGroupInvitation(groupID, reqBody.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Retrieve the notification based on notifyingUserID, notifiedUserID, and type
		notification, err := query.GetNotificationByDetails(currentUser.ID, reqBody.UserID, groupID, []string{"group_invitation"}, groupName)
		if err != nil {
			fmt.Printf("Error retrieving notification: %v", err)
			http.Error(w, "Failed to retrieve notification", http.StatusInternalServerError)
			return // Handle the error appropriately
		}
		if notification.ID != 0 {
			// Send the notification to the user via WebSocket
			websocket.SendDeNotificationToUser(appCore.Hub, reqBody.UserID) // Use the notification ID

			// Delete the notification
			_, err = query.DeleteInviteNotificationQuery(currentUser.ID, reqBody.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func GroupRequestHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		NotifyingUserId int    `json:"notifyingUserId"`
		NotifiedUserId  int    `json:"notifiedUserId"`
		GroupID         int    `json:"groupId"`
		Action          string `json:"action"`
	}

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	fmt.Println(request)
	// Implement logic to handle the Group request
	switch request.Action {
	case "accept":
		// Logic to accept the Group request
		err := query.AcceptGroupInvitation(request.GroupID, request.NotifiedUserId)
		if err != nil {
			http.Error(w, "Failed to accept grouprequest", http.StatusInternalServerError)
			return
		}
		err = query.ChangeNotificationType(request.NotifyingUserId, request.NotifiedUserId, []string{"group_invitation"}, "group")
		if err != nil {
			http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("group request accepted")
	case "decline":
		// Logic to decline the Group request
		err := query.RejectGroupInvitation(request.GroupID, request.NotifiedUserId)
		if err != nil {
			http.Error(w, "Failed to decline group request", http.StatusInternalServerError)
			return
		}
		err = query.ChangeNotificationType(request.NotifyingUserId, request.NotifiedUserId, []string{"group_invitation"}, "group")
		if err != nil {
			http.Error(w, "Couldn't Change Notification Type", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("group request declined")
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}

}

func GroupJoinRequestHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Group       models.Group `json:"group"`       // Group model
			RequestType string       `json:"requestType"` // Request type: "join" or "unjoin"
		}

		// Decode the request body
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		fmt.Println(request)

		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if request.RequestType == "join" {
			// Handle join request
			err = query.RequestAddGroupMember(int64(request.Group.ID), user.ID, "pending", request.Group.CreatorID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Create a notification for the creator
			notification := models.Notification{
				NotifiedUserID:  request.Group.CreatorID,
				NotifyingUserId: user.ID,
				Object:          request.Group.Name, // You can adjust this to be more descriptive
				ObjectID:        request.Group.ID,
				Type:            "group_join_request",
				Content:         user.Username + " requests to join '" + request.Group.Name + "' group",
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
			websocket.SendNotificationToUser(appCore.Hub, request.Group.CreatorID, notification)

		} else if request.RequestType == "unjoin" {
			// Handle unjoin request
			err = query.RemoveGroupMember(request.Group.ID, user.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Retrieve the existing notification related to the unjoin action
			notification, err := query.GetNotificationByDetails(user.ID, request.Group.CreatorID, request.Group.ID, []string{"group_join_request"}, request.Group.Name)
			if err != nil {
				fmt.Printf("Error retrieving notification: %v", err)
				http.Error(w, "Failed to retrieve notification", http.StatusInternalServerError)
				return
			}
			if notification.ID != 0 {
				// Check if the notification is unread
				if !notification.IsRead {
					// Send the de-notification to the user via WebSocket
					websocket.SendDeNotificationToUser(appCore.Hub, request.Group.CreatorID) // Use the notification ID
				}

				// Delete the existing notification
				err = query.DeleteNotificationQuery(notification.ID)
				if err != nil {
					http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
					return
				}
			}
		} else {
			http.Error(w, "Invalid request type", http.StatusBadRequest)
			return
		}
	}
}
