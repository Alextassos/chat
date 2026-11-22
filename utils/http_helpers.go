package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var errInvalidInput = errors.New("invalid input")

func trimAndNormalize(value string) string {
	return strings.TrimSpace(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
