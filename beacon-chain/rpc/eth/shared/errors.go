package shared

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/network"
)

func HandleHTTPError(w http.ResponseWriter, message string, code int) {
	errJson := &network.DefaultErrorJson{
		Message: message,
		Code:    code,
	}
	network.WriteError(w, errJson)
}
