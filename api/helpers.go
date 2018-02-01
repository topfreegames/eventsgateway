package api

import (
	"net/http"
)

type contextKey string

//Write to the response and with the status code
func Write(w http.ResponseWriter, status int, text string) {
	WriteBytes(w, status, []byte(text))
}

//WriteBytes to the response and with the status code
func WriteBytes(w http.ResponseWriter, status int, text []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(text)
}
