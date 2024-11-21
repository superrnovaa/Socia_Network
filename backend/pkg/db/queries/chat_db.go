package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"golang.org/x/exp/slices"
	"fmt"
	"log"
	//"slices"
	"strconv"
)

func CreateChatMessage(chatMessage models.ChatMessage) (models.ChatMessage, error) {
	query := `
		INSERT INTO messages (sender_id, receiver_id, group_id, content, created_at)
		VALUES (?, NULLIF(?, 0), NULLIF(?, 0), ?, ?);
	`
	result, err := sqlite.DB.Exec(query, chatMessage.SenderID, chatMessage.ReceiverID, chatMessage.GroupID, chatMessage.Content, chatMessage.CreatedAt)
	if err != nil {
		log.Printf("Error inserting message: %v\n%v", err, chatMessage)
		return models.ChatMessage{}, err
	}
	chatID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return models.ChatMessage{}, err
	}
	chatMessage.ID = int(chatID)
	return chatMessage, nil
}

func CreateChatNotifications(messageId int, notifiedUserIds []int) error {
	query := `
		INSERT INTO chat_notifications (message_id, notifiedUser_id)
		VALUES
	`
	for i, userId := range notifiedUserIds {
		if i > 0 {
			query += ","
		}
		query += fmt.Sprintf("(%v,%v)", messageId, userId)
	}
	query += ";"
	_, err := sqlite.DB.Exec(query)

	if err != nil {
		log.Printf("Error inserting message: %v\n%v", err, messageId, notifiedUserIds)
		return err
	}
	return nil
}

func DeleteChatNotifications(userId int, notifiedUserId int) error {
	query := `
		Delete FROM chat_notifications WHERE notifiedUser_id = ? AND message_id IN (
			SELECT id from messages WHERE sender_id=? AND group_id IS NULL
		)
	`
	_, err := sqlite.DB.Exec(query, userId, notifiedUserId)

	if err != nil {
		log.Printf("Error deleting notification: %v\n%v", err, userId, notifiedUserId)
		return err
	}
	return nil
}

func DeleteGroupChatNotifications(userId int, groupId int) error {
	query := `
		Delete FROM chat_notifications WHERE notifiedUser_id = ? AND message_id IN (
			SELECT id from messages WHERE group_id=?
		)
	`
	_, err := sqlite.DB.Exec(query, userId, groupId)

	if err != nil {
		log.Printf("Error deleting notification: %v\n%v", err, userId, groupId)
		return err
	}
	return nil
}

func GetChatQuery(userAId, userBId int) (models.Chat, error) {
	rows, err := sqlite.DB.Query("SELECT id, sender_id, content, created_at FROM messages WHERE (sender_id = ? AND receiver_id = ?) OR (receiver_id = ? AND sender_id  = ?) ORDER BY created_at ASC", userAId, userBId, userAId, userBId)
	if err != nil {
		log.Printf("Error retrieving chat: %v", err)
		return models.Chat{}, err
	}
	defer rows.Close()

	userA, err := GetUserItemByID(userAId)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		return models.Chat{}, err
	}
	userB, err := GetUserItemByID(userBId)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		return models.Chat{}, err
	}
	allowChat, err := CheckIfUsersFollowsOrFollowed(userAId, userBId)
	if err != nil {
		return models.Chat{}, err
	}
	chat := models.Chat{
		UserA:     userA,
		UserB:     userB,
		AllowChat: allowChat,
	}
	for rows.Next() {
		var n models.ChatMessage
		if err := rows.Scan(&n.ID, &n.SenderID, &n.Content, &n.CreatedAt); err != nil {
			log.Printf("Error scanning chat row: %v", err)
			return models.Chat{}, err
		}
		chat.Messages = append(chat.Messages, n)
	}
	return chat, nil
}

func GetGroupChatQuery(groupId int) (models.Chat, error) {
	rows, err := sqlite.DB.Query("SELECT id, sender_id, content, created_at FROM messages WHERE group_id = ? ORDER BY created_at ASC", groupId)
	if err != nil {
		log.Printf("Error retrieving chat: %v", err)
		return models.Chat{}, err
	}
	defer rows.Close()

	group, err := GetGroupData(groupId)
	if err != nil {
		log.Printf("Error retrieving group data: %v", err)
		return models.Chat{}, err
	}
	chat := models.Chat{
		Group: group,
	}
	for rows.Next() {
		var n models.ChatMessage
		if err := rows.Scan(&n.ID, &n.SenderID, &n.Content, &n.CreatedAt); err != nil {
			log.Printf("Error scanning chat row: %v", err)
			return models.Chat{}, err
		}
		chat.Messages = append(chat.Messages, n)
		if !slices.ContainsFunc(chat.Group.Members, func(e models.UserItem) bool { return e.ID == n.SenderID }) {
			addUser, err := GetUserItemByID(n.SenderID)
			if err != nil {
				log.Printf("Error retrieving deleted user data: %v", err)
				return models.Chat{}, err
			}
			chat.Group.Members = append(chat.Group.Members, addUser)
		}
	}

	return chat, nil
}

func GetAllChatQuery(userId int) ([]models.Chat, error) {
	rows, err := sqlite.DB.Query(`SELECT id, sender_id, receiver_id, group_id, content, created_at FROM messages 
	WHERE (sender_id = ? AND receiver_id IS NOT NULL) OR receiver_id = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ? AND status = "accepted") ORDER BY created_at DESC`, userId, userId, userId)
	if err != nil {
		log.Printf("Error retrieving chat: %v", err)
		return nil, err
	}
	defer rows.Close()

	userBIds := make(map[int]int)
	groupIds := make(map[int]int)
	var chats []models.Chat

	for rows.Next() {
		var message models.ChatMessage
		// Since GroupID and RecieverID might be NULL, I scan them into an []byte then process them later depending n which is nil
		var recieverIdarr []byte
		var groupIdarr []byte
		if err := rows.Scan(&message.ID, &message.SenderID, &recieverIdarr, &groupIdarr, &message.Content, &message.CreatedAt); err != nil {
			log.Printf("Error scanning chat row: %v", err)
			return nil, err
		}
		if groupIdarr != nil {
			// Chat Message
			message.GroupID, _ = strconv.Atoi(string(groupIdarr[:]))
			if _, found := groupIds[message.GroupID]; !found {
				group, err := GetGroupData(message.GroupID)
				if err != nil {
					log.Printf("Error retrieving group data: %v", err)
					return nil, err
				}

				notification, err := GetGroupChatNotificationsQuery(userId, message.GroupID)
				if err != nil {
					log.Printf("Error retrieving group data: %v", err)
					return nil, err
				}

				chats = append(chats, models.Chat{
					Group:        group,
					Notification: notification,
				})
				groupIds[message.GroupID] = len(chats) - 1
			}

			if !slices.ContainsFunc(chats[groupIds[message.GroupID]].Group.Members, func(e models.UserItem) bool { return e.ID == message.SenderID }) {
				addUser, err := GetUserItemByID(message.SenderID)
				if err != nil {
					log.Printf("Error retrieving deleted user data: %v", err)
					return nil, err
				}
				chats[groupIds[message.GroupID]].Group.Members = append(chats[groupIds[message.GroupID]].Group.Members, addUser)
			}

			chats[groupIds[message.GroupID]].Messages = append(chats[groupIds[message.GroupID]].Messages, message)
		} else {
			// Private Message
			message.ReceiverID, _ = strconv.Atoi(string(groupIdarr[:]))
			var userBId int
			// make UserAId always the currrent user
			if message.SenderID == userId {
				userBId = message.ReceiverID
			} else {
				userBId = message.SenderID
			}

			if _, found := userBIds[userBId]; !found {
				userA, err := GetUserItemByID(userId)
				if err != nil {
					log.Printf("Error retrieving user: %v", err)
					return nil, err
				}
				userB, err := GetUserItemByID(userBId)
				if err != nil {
					log.Printf("Error retrieving user: %v", err)
					return nil, err
				}

				notification, err := GetChatNotificationsQuery(userId, userBId)
				if err != nil {
					log.Printf("Error retrieving user data: %v", err)
					return nil, err
				}

				chats = append(chats, models.Chat{
					UserA:        userA,
					UserB:        userB,
					Notification: notification,
				})

				userBIds[userBId] = len(chats) - 1
			}
			chats[userBIds[userBId]].Messages = append(chats[userBIds[userBId]].Messages, message)
		}
	}
	return chats, nil
}

func GetLastMessageOfAllChatQuery(userId int) ([]models.Chat, error) {
	rows, err := sqlite.DB.Query(`SELECT id, sender_id, receiver_id, group_id, content, created_at FROM messages 
	WHERE (sender_id = ? AND receiver_id IS NOT NULL) OR receiver_id = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ? AND status = "accepted") ORDER BY created_at DESC`, userId, userId, userId)
	if err != nil {
		log.Printf("Error retrieving messages: %v", err)
		return nil, err
	}
	defer rows.Close()

	userBIds := make(map[int]int)
	groupIds := make(map[int]int)
	var chats []models.Chat

	for rows.Next() {
		var message models.ChatMessage
		// Since GroupID and RecieverID might be NULL, I scan them into an []byte then process them later depending n which is nil
		var receiverIdarr []byte
		var groupIdarr []byte
		if err := rows.Scan(&message.ID, &message.SenderID, &receiverIdarr, &groupIdarr, &message.Content, &message.CreatedAt); err != nil {
			log.Printf("Error scanning message row: %v", err)
			return nil, err
		}
		if groupIdarr != nil {
			// Chat Message
			message.GroupID, _ = strconv.Atoi(string(groupIdarr[:]))
			if _, found := groupIds[message.GroupID]; !found {
				group, err := GetGroupData(message.GroupID)
				if err != nil {
					log.Printf("Error retrieving group data: %v", err)
					return nil, err
				}

				if !slices.ContainsFunc(group.Members, func(e models.UserItem) bool { return e.ID == message.SenderID }) {
					addUser, err := GetUserItemByID(message.SenderID)
					if err != nil {
						log.Printf("Error retrieving deleted user data: %v", err)
						return nil, err
					}
					group.Members = append(group.Members, addUser)
				}

				notification, err := GetGroupChatNotificationsQuery(userId, message.GroupID)
				if err != nil {
					log.Printf("Error retrieving group data: %v", err)
					return nil, err
				}

				chats = append(chats, models.Chat{
					Group:        group,
					Notification: notification,
				})
				groupIds[message.GroupID] = len(chats) - 1

				chats[groupIds[message.GroupID]].Messages = append(chats[groupIds[message.GroupID]].Messages, message)
			}
		} else {
			// Private Message
			message.ReceiverID, _ = strconv.Atoi(string(receiverIdarr[:]))
			var userBId int
			// make UserAId always the currrent user
			if message.SenderID == userId {
				userBId = message.ReceiverID
			} else {
				userBId = message.SenderID
			}

			if _, found := userBIds[userBId]; !found {
				userA, err := GetUserItemByID(userId)
				if err != nil {
					log.Printf("Error retrieving userA: %v", err)
					return nil, err
				}
				userB, err := GetUserItemByID(userBId)
				if err != nil {
					log.Printf("Error retrieving userB %v %v: %v", groupIdarr, message.ReceiverID, err)
					return nil, err
				}

				notification, err := GetChatNotificationsQuery(userId, userBId)
				if err != nil {
					log.Printf("Error retrieving user data: %v", err)
					return nil, err
				}

				chats = append(chats, models.Chat{
					UserA:        userA,
					UserB:        userB,
					Notification: notification,
				})
				userBIds[userBId] = len(chats) - 1
				chats[userBIds[userBId]].Messages = append(chats[userBIds[userBId]].Messages, message)
			}
		}
	}
	return chats, nil
}

func GetChatNotificationsQuery(userId int, senderId int) (int, error) {
	var result int

	err := sqlite.DB.QueryRow(`SELECT count(*) AS Notifications 
	FROM chat_notifications AS n LEFT JOIN messages AS m ON n.message_id = m.id
	WHERE n.notifiedUser_id = ? AND m.sender_id = ? AND m.group_id IS NULL`, userId, senderId).Scan(&result)

	if err != nil {
		log.Printf("Error retrieving messages: %v", err)
		return 0, err
	}
	return result, nil
}

func GetGroupChatNotificationsQuery(userId int, groupId int) (int, error) {
	var result int

	err := sqlite.DB.QueryRow(`SELECT count(*) AS Notifications 
	FROM chat_notifications AS n LEFT JOIN messages AS m ON n.message_id = m.id
	WHERE n.notifiedUser_id = ? AND m.group_id = ? AND m.receiver_id IS NULL`, userId, groupId).Scan(&result)

	if err != nil {
		log.Printf("Error retrieving messages: %v", err)
		return 0, err
	}

	return result, nil
}

func GetNewChatUsers(userId int) ([]models.UserItem, error) {
	rows, err := sqlite.DB.Query(`SELECT id, username, avatar_url 
	FROM users 
	WHERE 
		id != ?
		AND id IN (
		SELECT follower_id as id FROM followers WHERE followed_id = ? 
		UNION
		SELECT followed_id as id FROM followers WHERE follower_id = ? 
		) 
		AND id NOT IN(
		SELECT sender_id as id FROM messages WHERE receiver_id = ?
		UNION
		SELECT receiver_id as id FROM messages WHERE sender_id = ? AND receiver_id IS NOT NULL
		)`, userId, userId, userId, userId, userId)
	if err != nil {
		log.Printf("Error retrieving users: %v", err)
		return nil, err
	}

	var users []models.UserItem
	for rows.Next() {
		var user models.UserItem
		if err := rows.Scan(&user.ID, &user.Username, &user.ProfileImg); err != nil {
			log.Printf("Error scanning user row: %v", err)
			return nil, err
		}
		users = append(users, user)
	}
	defer rows.Close()

	return users, nil
}
