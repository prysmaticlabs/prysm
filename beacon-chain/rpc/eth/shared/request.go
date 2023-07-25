package shared

import (
	"net/http"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network"
)

func ValidateHex(w http.ResponseWriter, name string, s string) bool {
	if s == "" {
		errJson := &network.DefaultErrorJson{
			Message: name + " is required",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return false
	}
	if !bytesutil.IsHex([]byte(s)) {
		errJson := &network.DefaultErrorJson{
			Message: name + " is invalid",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return false
	}
	return true
}

func ValidateUint(w http.ResponseWriter, name string, s string) (uint64, bool) {
	if s == "" {
		errJson := &network.DefaultErrorJson{
			Message: name + " is required",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: name + " is invalid: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return 0, false
	}
	return v, true
}
