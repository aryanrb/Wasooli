package httperr

import "net/http"

// Error carries an HTTP status alongside a client-safe message. Services return
// these; the server's error handler renders them as JSON and logs 5xx causes.
type Error struct {
	Status  int
	Message string
	Err     error // underlying cause, never sent to the client
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func New(status int, msg string) *Error { return &Error{Status: status, Message: msg} }

func BadRequest(msg string) *Error   { return New(http.StatusBadRequest, msg) }
func Unauthorized(msg string) *Error { return New(http.StatusUnauthorized, msg) }
func Forbidden(msg string) *Error    { return New(http.StatusForbidden, msg) }
func NotFound(msg string) *Error     { return New(http.StatusNotFound, msg) }
func Conflict(msg string) *Error     { return New(http.StatusConflict, msg) }

// Internal wraps an unexpected error; the cause is logged, not returned.
func Internal(err error) *Error {
	return &Error{Status: http.StatusInternalServerError, Message: "internal server error", Err: err}
}
