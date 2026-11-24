package httpapi

import (
	"encoding/json"
	"net/http"
)

// errorBody is the single error shape returned by the API:
// {"error": {"code": "SEAT_TAKEN", "message": "..."}}
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends a client-safe error. Never pass internal error text as
// message: log it instead and send a generic description.
func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}
