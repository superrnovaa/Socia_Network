package utilities

import (
	"backend/pkg/db/sqlite"
	"backend/pkg/models"
	"errors"
	"regexp"
	"strings"
	"time"
)

func ValidateUser(data models.User) error {
	// Email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,4}$`)
	if !emailRegex.MatchString(data.Email) {
		return errors.New("Invalid email format")
	}

	// Check if email already exists
	var count int
	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM users WHERE Email = ?", data.Email).Scan(&count)
	if err != nil {
		return errors.New("Error checking email existence")
	}
	if count > 0 {
		return errors.New("Email already exists")
	}

	// Username validation: allow letters, numbers, and some special characters
	if !isValidName(data.Username) {
		return errors.New("Username must contain only letters, numbers, and special characters")
	}

	// Check if username already exists
	err = sqlite.DB.QueryRow("SELECT COUNT(*) FROM users WHERE Username = ?", data.Username).Scan(&count)
	if err != nil {
		return errors.New("Error checking username existence")
	}
	if count > 0 {
		return errors.New("Username already exists")
	}

	// Nickname validation: optional, but if provided, must be valid
	if data.Nickname != "" && !isValidName(data.Nickname) {
		return errors.New("Nickname must contain only letters, numbers, and special characters")
	}

	// Name validation: allow letters and spaces
	if !isValidName(data.FirstName) {
		return errors.New("First name must contain only letters, numbers, and special characters")
	}

	if !isValidName(data.LastName) {
		return errors.New("Last name must contain only letters, numbers, and special characters")
	}

	// Age validation
	birthDate, err := time.Parse("2006-01-02", data.DateOfBirth)
	if err != nil {
		return errors.New("Invalid date of birth format. Use YYYY-MM-DD")
	}

	age := time.Since(birthDate).Hours() / 24 / 365.25
	if age < 18 {
		return errors.New("User must be at least 18 years old")
	}

	return nil
}

func isValidName(s string) bool {
	// Use a regular expression to match only Latin letters, numbers, underscores, and hyphens
	validUsernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validUsernameRegex.MatchString(s)
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(email)
}

func sanitizeInput(input string) string {
	// Remove leading and trailing whitespace
	input = strings.TrimSpace(input)

	// Remove special characters
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	input = re.ReplaceAllString(input, "")

	return input
}
