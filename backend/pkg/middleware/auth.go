package middleware

import (
	query "backend/pkg/db/queries"
	"backend/pkg/models"
	"context"
	"errors"
	"net/http"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := GetAuthenticatedUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add the user to the request context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetAuthenticatedUser retrieves the authenticated user from the session
func GetAuthenticatedUser(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	user, err := query.GetSessionUser(cookie.Value)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, http.ErrNoCookie
	}

	return user, nil
}

// GetUserFromContext retrieves the authenticated user from the request context
func GetUserFromContext(r *http.Request) (*models.User, error) {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return nil, errors.New("no authenticated user found in context")
	}
	return user, nil
}
