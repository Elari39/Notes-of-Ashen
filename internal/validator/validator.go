package validator

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	apperrors "notes-of-ashen/internal/errors"
)

func Required(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return apperrors.BadRequest(field + " is required")
	}
	return nil
}

func Length(value, field string, min, max int) error {
	n := utf8.RuneCountInString(value)
	if n < min || n > max {
		return apperrors.BadRequest(field + " length is invalid")
	}
	return nil
}

func Email(value string) error {
	if _, err := mail.ParseAddress(value); err != nil {
		return apperrors.BadRequest("email format is invalid")
	}
	return nil
}

func OptionalHTTPURL(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return apperrors.BadRequest(field + " format is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return apperrors.BadRequest(field + " format is invalid")
	}
	return nil
}

func Status(value string, allowed map[string]struct{}, field string) error {
	if _, ok := allowed[value]; !ok {
		return apperrors.BadRequest(field + " is invalid")
	}
	return nil
}
