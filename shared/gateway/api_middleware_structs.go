package gateway

// ---------------
// Error handling.
// ---------------

// ErrorJson describes common functionality of all JSON error representations.
type ErrorJson interface {
	StatusCode() int
	SetCode(code int)
	Msg() string
}

// DefaultErrorJson is a JSON representation of a simple error value, containing only a message and an error code.
type DefaultErrorJson struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// StatusCode returns the error's underlying error code.
func (e *DefaultErrorJson) StatusCode() int {
	return e.Code
}

// Msg returns the error's underlying message.
func (e *DefaultErrorJson) Msg() string {
	return e.Message
}

// SetCode sets the error's underlying error code.
func (e *DefaultErrorJson) SetCode(code int) {
	e.Code = code
}
