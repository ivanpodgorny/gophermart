package handler

import (
	"encoding/json"
	"net/http"
)

func badRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

func serverError(w http.ResponseWriter) {
	http.Error(w, "500 internal server error", http.StatusInternalServerError)
}

func responseAsJSON(w http.ResponseWriter, v any, code int) {
	respJSON, err := json.Marshal(v)
	if err != nil {
		serverError(w)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(respJSON); err != nil {
		serverError(w)
	}
}
