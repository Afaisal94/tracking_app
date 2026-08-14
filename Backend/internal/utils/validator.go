package utils

import (
	"net/mail"
	"strings"
)

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

func ValidatePassword(pw string) (bool, string) {
	if len(pw) < 8 {
		return false, "password must be at least 8 characters"
	}
	return true, ""
}
