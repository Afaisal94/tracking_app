package utils

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, status int, message string, data interface{}) {
	JSON(w, status, APIResponse{Success: true, Message: message, Data: data})
}

func SuccessWithMeta(w http.ResponseWriter, status int, message string, data interface{}, meta interface{}) {
	JSON(w, status, APIResponse{Success: true, Message: message, Data: data, Meta: meta})
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIResponse{Success: false, Error: message})
}

func ValidationError(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}
