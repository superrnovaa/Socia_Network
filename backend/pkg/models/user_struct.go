package models

import (
	"time"
)

type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Password      string    `json:"-"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Nickname      string    `json:"nickname,omitempty"`
	DateOfBirth   string    `json:"dateOfBirth"`
	AboutMe       string    `json:"aboutMe"`
	AvatarURL     string    `json:"avatarUrl,omitempty"`
	IsPublic      bool      `json:"isPublic"`
	CreatedAt     time.Time `json:"createdAt"`
	Following     int       `json:"following"`
	Followers     int       `json:"followers"`
	FollowState   string    `json:"followState"`
	Notifications int       `json:"notifications"`
	PostCount     int       `json:"postCount"`
}

var UserInfo *User

type UserItem struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	ProfileImg string `json:"profileImg"`
	PostCount  int    `json:"postCount"`
}

// SafeUser returns a copy of the User without sensitive information
func (u *User) SafeUser() map[string]interface{} {
	return map[string]interface{}{
		"id":            u.ID,
		"username":      u.Username,
		"email":         u.Email,
		"firstName":     u.FirstName,
		"lastName":      u.LastName,
		"nickname":      u.Nickname,
		"dateOfBirth":   u.DateOfBirth,
		"aboutMe":       u.AboutMe,
		"avatarUrl":     u.AvatarURL,
		"createdAt":     u.CreatedAt,
		"following":     u.Following,
		"followers":     u.Followers,
		"followState":   u.FollowState,
		"notifications": u.Notifications,
		"isPublic":      u.IsPublic,
	}
}

// FollowRequest represents the structure of the follow request
type FollowRequest struct {
	FollowerID  int    `json:"followerId"`  // ID of the profile being followed/unfollowed
	FollowedID  int    `json:"followedId"`  // ID of the user performing the action
	ButtonState string `json:"buttonState"` // New button state (Follow, Unfollow, Pending)
}
