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

func FeatureNotImplemented(featureKey string) *Error {
	return ErrorWithDetails(501, "feature_not_implemented", "This open-source feature is not implemented yet", map[string]string{
		"featureKey": featureKey,
	})
}
