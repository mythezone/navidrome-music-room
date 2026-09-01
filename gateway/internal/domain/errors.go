package domain

import "fmt"

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func ErrorWithDetails(status int, code, message string, details any) *Error {
	return &Error{Status: status, Code: code, Message: message, Details: details}
}

func FeatureLocked(featureKey string) *Error {
	return ErrorWithDetails(402, "feature_locked", "This feature requires a license", map[string]string{
		"featureKey": featureKey,
	})
}
