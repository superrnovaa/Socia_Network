package api

import (
	query "backend/pkg/db/queries"
	"backend/pkg/middleware"
	"backend/pkg/models"
	"backend/pkg/websocket"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func InitWebSocketConnectionHandler(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	}
}

func GetAllChatsHandler(w http.ResponseWriter, r *http.Request) {
	// Right now this only return the latest message in the chat because I do not see the value in sending everything.
	// If for some reason you want everything, uncomment the GetAllChatQuery
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch notifications from the database
	//chats, err := query.GetAllChatQuery(user.ID) // Probably do a url variable to toggle this on
	chats, err := query.GetLastMessageOfAllChatQuery(user.ID)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
		return
	}

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode notifications to JSON and send the response
	if err := json.NewEncoder(w).Encode(chats); err != nil {
		http.Error(w, "Failed to encode chats", http.StatusInternalServerError)
	}
}

func GetChatHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userBName := r.URL.Query().Get("userBName")
	if userBName == "" {
		http.Error(w, "Invalid User Name", http.StatusBadRequest)
		return
	}
	userBId, err := query.GetUserIdByUsername(userBName)
	if err != nil {
		http.Error(w, "Failed to find user", http.StatusInternalServerError)
		return
	}
	// Fetch chat from the database
	chat, err := query.GetChatQuery(user.ID, userBId)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
		return
	}

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode chat to JSON and send the response
	if err := json.NewEncoder(w).Encode(chat); err != nil {
		http.Error(w, "Failed to encode chats", http.StatusInternalServerError)
	}
}

func GetGroupChatHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupId, err := strconv.Atoi(r.URL.Query().Get("groupId"))
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	// Fetch notifications from the database
	chat, err := query.GetGroupChatQuery(groupId)
	if err != nil {
		http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
		return
	}
	userA, err := query.GetUserItemByID(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
		return
	}
	chat.UserA = userA

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode notifications to JSON and send the response
	if err := json.NewEncoder(w).Encode(chat); err != nil {
		http.Error(w, "Failed to encode chats", http.StatusInternalServerError)
	}
}

func SendMessageHandler(appCore *middleware.AppCore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := middleware.GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var message models.ChatMessage
		err = json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		// Verification
		if user.ID != message.SenderID {
			http.Error(w, "Sender is not the same as the logged in User", http.StatusBadRequest)
			return
		}
		if message.GroupID != 0 {
			status, err := query.GetMemberStatus(message.SenderID, message.GroupID)
			if err != nil {
				http.Error(w, "Failed to check user group member status", http.StatusInternalServerError)
				return
			}
			if status != "accepted" {
				http.Error(w, "Sender is not part of the group", http.StatusBadRequest)
				return
			}
		} else {
			allowChat, err := query.CheckIfUsersFollowsOrFollowed(message.SenderID, message.ReceiverID)
			if err != nil {
				http.Error(w, "Failed to check user follower status", http.StatusInternalServerError)
				return
			}
			if !allowChat {
				http.Error(w, "Users do not have a follower relationship", http.StatusBadRequest)
				return
			}
		}
		// Create and send the message
		message, err = query.CreateChatMessage(message)
		if err != nil {
			http.Error(w, "Failed to add message", http.StatusInternalServerError)
			return
		}
		if message.GroupID == 0 {
			websocket.SendChatToUsers(appCore.Hub, message)
		} else {
			websocket.SendGroupChatToUsers(appCore.Hub, message)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(message); err != nil {
			http.Error(w, "Error encoding new comment to JSON", http.StatusInternalServerError)
			return
		}
	}
}

func GetNewChatUsersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	users, err := query.GetNewChatUsers(user.ID)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, "Failed to fetch chat users", http.StatusInternalServerError)
		return
	}

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode notifications to JSON and send the response
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, "Failed to encode chat users", http.StatusInternalServerError)
	}
}

func MarkMessageAsReadHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetAuthenticatedUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userBName := r.URL.Query().Get("userBName")
	groupId, err := strconv.Atoi(r.URL.Query().Get("groupId"))
	if userBName == "" && (err != nil || groupId == 0) {
		http.Error(w, "Invalid Username/groupId", http.StatusBadRequest)
		return
	}
	if userBName != "" {
		userBId, err := query.GetUserIdByUsername(userBName)
		if err != nil {
			http.Error(w, "Failed to find user", http.StatusInternalServerError)
		}

		err = query.DeleteChatNotifications(user.ID, userBId)
		if err != nil {
			http.Error(w, "Failed to mark chat as read", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		err = query.DeleteGroupChatNotifications(user.ID, groupId)
		if err != nil {
			http.Error(w, "Failed to mark chat as read", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
