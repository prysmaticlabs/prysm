package httputil

import (
	"net/http"
)

func HandleError(w http.ResponseWriter, message string, code int) {
	errJson := &DefaultErrorJson{
		Message: message,
		Code:    code,
	}
	WriteError(w, errJson)
}
