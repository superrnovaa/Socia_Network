package query

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"database/sql"
	"errors"
	"log"
	"fmt"
	"strings"
)

func GetGroupByName(name string) (models.Group, error) {
	var group models.Group
	query := `SELECT id, name, description, creator_id, image_url, created_at 
              FROM groups 
              WHERE name = ? OR name = ?`

	urlSafeName := strings.ReplaceAll(name, " ", "-")

	err := sqlite.DB.QueryRow(query, name, urlSafeName).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CreatorID,
		&group.ImageURL,
		&group.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Group{}, errors.New("group not found")
		}
		log.Printf("Error retrieving group by name: %v", err)
		return models.Group{}, err
	}

	// Fetch group members with status 'accepted'
	memberQuery := `SELECT u.id, u.username, u.avatar_url 
                    FROM group_members gm 
                    JOIN users u ON gm.user_id = u.id 
                    WHERE gm.group_id = ? AND gm.status = 'accepted'
                    ORDER BY CASE WHEN gm.user_id = ? THEN 0 ELSE 1 END, gm.created_at`
	rows, err := sqlite.DB.Query(memberQuery, group.ID, group.CreatorID)
	if err != nil {
		log.Printf("Error retrieving group members: %v", err)
		return models.Group{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var member models.UserItem
		if err := rows.Scan(&member.ID, &member.Username, &member.ProfileImg); err != nil {
			log.Printf("Error scanning group member: %v", err)
			return models.Group{}, err
		}
		group.Members = append(group.Members, member)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over group members: %v", err)
		return models.Group{}, err
	}

	return group, nil
}

func CreateGroupQuery(newGroup models.Group) (int64, error) { // Change return type to (int64, error)
	result, err := sqlite.DB.Exec("INSERT INTO groups (name, description, creator_id, image_url) VALUES (?, ?, ?, ?)",
		newGroup.Name, newGroup.Description, newGroup.CreatorID, newGroup.ImageURL)
	if err != nil {
		log.Printf("Error creating group: %v", err)
		return 0, err // Return 0 and the error
	}

	groupID, err := result.LastInsertId() // Get the last inserted ID
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		return 0, err // Return 0 and the error
	}

	return groupID, nil // Return the group ID and nil error
}

func IsCreator(groupID, userID int) (bool, error) {
	creatorID, err := GetGroupCreatorID(groupID) // New function to get creator ID
	if err != nil {
		return false, err
	}

	if creatorID != userID {
		return false, nil
	}

	return true, nil
}

// CheckGroupNameExists checks if a group name already exists in the database
func CheckGroupNameExists(groupName string) (bool, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM groups WHERE name = ?", groupName).Scan(&count)
	if err != nil {
		log.Printf("Error checking group name existence: %v", err)
		return false, err
	}
	return count > 0, nil // Return true if count is greater than 0
}

func GetGroupQuery(groupID int) (models.Group, error) {
	var g models.Group

	// SQL query to select a group by its ID
	query := "SELECT id, name, description, creator_id, image_url, created_at FROM groups WHERE id = ?"

	// Execute the query
	row := sqlite.DB.QueryRow(query, groupID)

	// Scan the result into the group variable
	if err := row.Scan(&g.ID, &g.Name, &g.Description, &g.CreatorID, &g.ImageURL, &g.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return models.Group{}, nil // Return nil if no group is found
		}
		log.Printf("Error scanning group row: %v", err)
		return models.Group{}, err
	}

	return g, nil // Return a pointer to the group
}

func UpdateGroupQuery(updatedGroup models.Group) error {
	query := "UPDATE groups SET name = ?, description = ?"
	params := []interface{}{updatedGroup.Name, updatedGroup.Description}

	// Check if image_url is not empty and append to query and params
	if updatedGroup.ImageURL != "" {
		query += ", image_url = ?"
		params = append(params, updatedGroup.ImageURL)
	}

	query += " WHERE id = ?"
	params = append(params, updatedGroup.ID)

	_, err := sqlite.DB.Exec(query, params...)
	if err != nil {
		log.Printf("Error updating group: %v", err)
		return err
	}
	return nil
}

/*

func DeleteGroupQuery(groupID int) error {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}

	// Delete the group
	_, err = tx.Exec("DELETE FROM groups WHERE id = ?", groupID)
	if err != nil {
		tx.Rollback()
		log.Printf("Error deleting group: %v", err)
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	return nil
}
	*/


	func DeleteGroupQuery(groupID int) error {
		// Delete notifications related to the group
		_, err := sqlite.DB.Exec(`
			DELETE FROM notifications 
			WHERE object_id = ? 
			AND type IN ('group', 'group_invitation', 'group_join_request')`, groupID)
		if err != nil {
			log.Printf("Error deleting notifications for group ID %d: %v", groupID, err)
			return err
		}
	
		// Delete notifications for posts that belong to the group
		_, err = sqlite.DB.Exec(`
			DELETE FROM notifications 
			WHERE object_id IN (
				SELECT id FROM posts WHERE group_id = ?
			) 
			AND type IN ('comment', 'reaction', 'post')`, groupID)
		if err != nil {
			log.Printf("Error deleting notifications for posts in group ID %d: %v", groupID, err)
			return err
		}

		    // Delete notifications of type 'event' with object_id of groupID
			_, err = sqlite.DB.Exec(`
			DELETE FROM notifications 
			WHERE object_id = ? 
			AND type = 'event'`, groupID)
		if err != nil {
			log.Printf("Error deleting notifications of type 'event' for group ID %d: %v", groupID, err)
			return err
		}
	
	
		// Delete the group
		_, err = sqlite.DB.Exec("DELETE FROM groups WHERE id = ?", groupID)
		if err != nil {
			log.Printf("Error deleting group: %v", err)
			return err
		}
	
		return nil
	}

// GetPostsByGroupID retrieves all posts for a specific group
func GetPostsByGroupID(groupID int) ([]models.Post, error) {
	rows, err := sqlite.DB.Query(`
        SELECT p.id, p.title, p.content, p.image_url, p.privacy, p.created_at,
               u.id AS user_id, u.username, u.avatar_url
        FROM posts p
        JOIN users u ON p.user_id = u.id
        WHERE p.group_id = ?
        ORDER BY p.created_at DESC
    `, groupID)
	if err != nil {
		log.Printf("Error retrieving posts for group ID %d: %v", groupID, err)
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var safeUser models.SafeUser
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.File, &p.Privacy, &p.CreatedAt,
			&safeUser.ID, &safeUser.Username, &safeUser.AvatarURL,
		); err != nil {
			log.Printf("Error scanning post row: %v", err)
			return nil, err
		}
		p.User = safeUser
		posts = append(posts, p)
	}

	return posts, nil
}



func RequestAddGroupMember(groupID int64, userID int, status string, inviterID int) error {
	// Check if there is a pending record for the user in the group
	var existingStatus string
	err := sqlite.DB.QueryRow("SELECT status FROM group_members WHERE group_id = ? AND user_id = ? AND status = 'pending'", groupID, userID).Scan(&existingStatus)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error checking for existing pending group member: %v", err)
		return err
	}

	// If there is a pending record, delete it before adding the new record
	if existingStatus == "pending" {
		_, err = sqlite.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ? AND status = 'pending'", groupID, userID)
		if err != nil {
			log.Printf("Error deleting pending record for user ID %d in group ID %d: %v", userID, groupID, err)
			return err
		}
	} else {
		// Insert a new row for the user in the group if there is no pending record
		_, err = sqlite.DB.Exec("INSERT INTO group_members (group_id, user_id, inviter_id, status) VALUES (?, ?, ?, ?)", groupID, userID, inviterID, status)
		if err != nil {
			log.Printf("Error adding user ID %d to group ID %d: %v", userID, groupID, err)
			return err
		}
	}

	return nil
}

func GetGroupData(groupID int) (models.Group, error) {
	group, err := GetGroupQuery(groupID)
	if err != nil {
		return models.Group{}, err
	}

	group.Members, err = GetGroupMembersAsUsers(groupID)
	if err != nil {
		return models.Group{}, err
	}

	return group, nil
}

func GetGroupsNotInUserMembership(userID int) ([]models.Group, error) {
	// Prepare the SQL query to get group IDs where the user is not a member with accepted status
	rows, err := sqlite.DB.Query(`
        SELECT g.id, g.name, g.image_url
        FROM groups g
        WHERE g.id NOT IN (
            SELECT gm.group_id
            FROM group_members gm
            WHERE gm.user_id = ? AND gm.status = 'accepted'
        )
        OR g.id IN (
            SELECT gm.group_id
            FROM group_members gm
            WHERE gm.user_id = ? AND gm.status = 'pending'
        )`, userID, userID)

	if err != nil {
		log.Printf("Error retrieving groups for user ID %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	// Iterate through the result set
	for rows.Next() {
		var group models.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.ImageURL); err != nil {
			log.Printf("Error scanning group: %v", err)
			return nil, err
		}
		groups = append(groups, group) // Append the retrieved group to the slice
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error during rows iteration: %v", err)
		return nil, err
	}

	return groups, nil // Return the slice of Group structs
}

func GetGroupMembersAsUsers(groupID int) ([]models.UserItem, error) {
	// Query to get user IDs of accepted group members

	rows, err := sqlite.DB.Query(`
        SELECT user_id
        FROM group_members
        WHERE group_id = ? AND status = ?`, groupID, "accepted")

	if err != nil {
		log.Printf("Error retrieving user IDs for group ID %d: %v", groupID, err)
		return nil, err
	}
	defer rows.Close()

	var userIDs []int
	// Collect user IDs
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			log.Printf("Error scanning user ID: %v", err)
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error during rows iteration: %v", err)
		return nil, err
	}

	// Now retrieve user details for each user ID
	var users []models.UserItem
	for _, userID := range userIDs {
		user, err := GetUserItemByID(userID) 
		if err != nil {
			log.Printf("Error retrieving user details for user ID %d: %v", userID, err)
			continue // Skip this user if there's an error
		}
		users = append(users, user) // Append the retrieved user to the slice
	}

	return users, nil // Return the slice of UserItem
}

// AddGroupMember adds a new member to a group
func AddGroupMember(groupID int64, userID int, status string, inviterID int) error {
	_, err := sqlite.DB.Exec("INSERT INTO group_members (group_id, user_id, inviter_id, status) VALUES (?, ?, ?, ?)", groupID, userID, inviterID, status)
	if err != nil {
		log.Printf("Error adding user ID %d to group ID %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

// RemoveGroupMember removes a member from a group
func RemoveGroupMember(groupID, userID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
	if err != nil {
		log.Printf("Error removing user ID %d from group ID %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

// InviteUsersToGroup invites multiple users to a group
func InviteUsersToGroup(groupID int, userIDs []int, inviterID int) error { // Accept inviterID as a parameter
	for _, userID := range userIDs {
		_, err := sqlite.DB.Exec("INSERT INTO group_members (group_id, user_id, status, inviter_id) VALUES (?, ?, 'pending', ?)", groupID, userID, inviterID)
		if err != nil {
			log.Printf("Error inviting user ID %d to group ID %d: %v", userID, groupID, err)
			return err
		}
	}
	return nil
}

// CancelGroupInvitation removes a user's invitation to a group
func CancelGroupInvitation(groupID, userID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ? AND status = 'pending'", groupID, userID)
	if err != nil {
		log.Printf("Error canceling invitation for user ID %d from group ID %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

// AcceptGroupInvitation accepts a group invitation for a user
func AcceptGroupInvitation(groupID, userID int) error {
	_, err := sqlite.DB.Exec("UPDATE group_members SET status = 'accepted' WHERE group_id = ? AND user_id = ? AND status = 'pending'", groupID, userID)
	if err != nil {
		log.Printf("Error accepting invitation for user ID %d to group ID %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

// RejectGroupInvitation rejects a group invitation for a user
func RejectGroupInvitation(groupID, userID int) error {
	_, err := sqlite.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ? AND status = 'pending'", groupID, userID)
	if err != nil {
		log.Printf("Error rejecting invitation for user ID %d to group ID %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

// RemoveGroupMembers removes multiple members from a group
func RemoveGroupMembers(groupID int, userIDs []int) error {
	for _, userID := range userIDs {
		_, err := sqlite.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
		if err != nil {
			log.Printf("Error removing user ID %d from group ID %d: %v", userID, groupID, err)
			return err
		}
	}
	return nil
}

// IsUserInGroup checks if a user is already a member of the group
func IsUserInGroup(groupID, userID int) (bool, error) {
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID).Scan(&count)
	if err != nil {
		log.Printf("Error checking if user ID %d is in group ID %d: %v", userID, groupID, err)
		return false, err
	}
	return count > 0, nil
}

// GetGroupCreatorID retrieves the creator ID of a specific group
func GetGroupCreatorID(groupID int) (int, error) {
	var creatorID int
	err := sqlite.DB.QueryRow("SELECT creator_id FROM groups WHERE id = ?", groupID).Scan(&creatorID)
	if err != nil {
		log.Printf("Error retrieving creator ID for group ID %d: %v", groupID, err)
		return 0, err
	}
	return creatorID, nil
}

// GetGroupIDByName retrieves the group ID based on the group name
func GetGroupIDByName(groupName string) (int, error) {
	var groupID int

	// Prepare the SQL query
	query := "SELECT id FROM groups WHERE name = ?"
	err := sqlite.DB.QueryRow(query, groupName).Scan(&groupID)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("group not found")
		}
		return 0, err // Return any other error
	}

	return groupID, nil // Return the found group ID
}

// getMemberStatus checks if a user is a member of the group and returns their status
func GetMemberStatus(userID, groupID int) (string, error) {
	var status string

	// SQL query to get the member status
	query := `
        SELECT status 
        FROM group_members 
        WHERE user_id = ? AND group_id = ?`

	// Execute the query
	err := sqlite.DB.QueryRow(query, userID, groupID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User is not a member, return empty string
		}
		return "", err // Return empty string on error
	}

	return status, nil // Return the status if found
}

func GetCategorizedGroups(userID int) (map[string][]models.Group, error) {
	categorizedGroups := map[string][]models.Group{
		"Created":  {},
		"Joined":   {},
		"Discover": {},
	}

	// Get Created groups
	createdGroups, err := getCreatedGroups(userID)
	if err != nil {
		return nil, err
	}
	categorizedGroups["Created"] = createdGroups

	// Get Joined groups
	joinedGroups, err := getJoinedGroups(userID)
	if err != nil {
		return nil, err
	}
	categorizedGroups["Joined"] = joinedGroups

	// Get Discover groups
	discoverGroups, err := getDiscoverGroups(userID)
	if err != nil {
		return nil, err
	}
	categorizedGroups["Discover"] = discoverGroups

	return categorizedGroups, nil
}

func getCreatedGroups(userID int) ([]models.Group, error) {
	query := `
		SELECT id, name, description, creator_id, image_url, created_at
		FROM groups
		WHERE creator_id = ?
	`
	return executeGroupQuery(query, userID)
}

func getJoinedGroups(userID int) ([]models.Group, error) {
	query := `
		SELECT g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ? AND gm.status = 'accepted' AND g.creator_id != ?
	`
	return executeGroupQuery(query, userID, userID)
}

func getDiscoverGroups(userID int) ([]models.Group, error) {
	query := `
		SELECT g.id, g.name, g.description, g.creator_id, g.image_url, g.created_at
		FROM groups g
		WHERE g.id NOT IN (
			SELECT group_id
			FROM group_members
			WHERE user_id = ? AND status = 'accepted'
		) AND g.creator_id != ?
	`
	return executeGroupQuery(query, userID, userID)
}

func executeGroupQuery(query string, args ...interface{}) ([]models.Group, error) {
	rows, err := sqlite.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatorID, &g.ImageURL, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, nil
}


// GetNotificationsByDetailsWithoutUser retrieves notifications for a specific groupID and types without filtering by notifying user ID.
func GetNotificationsByGroupID(groupID int, types []string) ([]models.Notification, error) {
    // Construct the SQL query to fetch notifications for the given groupID and types
    // Create a placeholder string for the types
    placeholders := make([]string, len(types))
    for i := range types {
        placeholders[i] = "?"
    }
    placeholderString := strings.Join(placeholders, ", ")

    query := fmt.Sprintf(`
        SELECT * FROM notifications 
        WHERE object_id = ? 
        AND type IN (%s)`, placeholderString)

    // Prepare the arguments for the query
    args := []interface{}{groupID}
    for _, t := range types {
        args = append(args, t)
    }

    // Execute the query
    rows, err := sqlite.DB.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Slice to hold the notifications
    var notifications []models.Notification

    // Iterate through the rows and scan the results into the notifications slice
    for rows.Next() {
        var notification models.Notification
        if err := rows.Scan(&notification.ID, &notification.NotifiedUserID, &notification.NotifyingUserId, &notification.Object, &notification.ObjectID, &notification.Type, &notification.Content, &notification.IsRead, &notification.CreatedAt, &notification.NotifyingImage); err != nil {
            return nil, err
        }
        notifications = append(notifications, notification)
    }

    // Check for errors from iterating over rows
    if err := rows.Err(); err != nil {
        return nil, err
    }

    return notifications, nil
}
