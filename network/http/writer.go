package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	jsonMediaType        = "application/json"
	octetStreamMediaType = "application/octet-stream"
)

// DefaultErrorJson is a JSON representation of a simple error value, containing only a message and an error code.
type DefaultErrorJson struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// WriteJson writes the response message in JSON format.
func WriteJson(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", jsonMediaType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.WithError(err).Error("Could not write response message")
	}
}

// WriteSsz writes the response message in ssz format
func WriteSsz(w http.ResponseWriter, respSsz []byte, fileName string) {
	w.Header().Set("Content-Length", strconv.Itoa(len(respSsz)))
	w.Header().Set("Content-Type", octetStreamMediaType)
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	if _, err := io.Copy(w, io.NopCloser(bytes.NewReader(respSsz))); err != nil {
		log.WithError(err).Error("could not write response message")
	}
}

// WriteError writes the error by manipulating headers and the body of the final response.
func WriteError(w http.ResponseWriter, errJson *DefaultErrorJson) {
	j, err := json.Marshal(errJson)
	if err != nil {
		log.WithError(err).Error("Could not marshal error message")
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(j)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errJson.Code)
	if _, err := io.Copy(w, io.NopCloser(bytes.NewReader(j))); err != nil {
		log.WithError(err).Error("Could not write error message")
	}
}
