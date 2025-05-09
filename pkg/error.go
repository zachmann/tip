package pkg

import (
	"fmt"
)

type TIPError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	Status           int    `json:"-"`
}

// Error implements the error interface
func (e TIPError) Error() string {
	s := e.ErrorCode
	if e.ErrorDescription != "" {
		s = fmt.Sprintf("%s: %s", s, e.ErrorDescription)
	}
	return s
}

func invalidRequestError(description string) TIPError {
	return TIPError{
		ErrorCode:        "invalid_request",
		ErrorDescription: description,
		Status:           400,
	}
}

func invalidClientError(description string) TIPError {
	return TIPError{
		ErrorCode:        "invalid_client",
		ErrorDescription: description,
		Status:           401,
	}
}

func unauthorizedError(description string) TIPError {
	return TIPError{
		ErrorCode:        "unauthorized_client",
		ErrorDescription: description,
		Status:           401,
	}
}

func notFoundError(description string) TIPError {
	return TIPError{
		ErrorCode:        "not_found",
		ErrorDescription: description,
		Status:           404,
	}
}

func internalServerError(description string) TIPError {
	return TIPError{
		ErrorCode:        "internal_server_error",
		ErrorDescription: description,
		Status:           500,
	}
}
