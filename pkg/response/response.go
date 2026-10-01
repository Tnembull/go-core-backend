package response

import (
	"encoding/json"
	"net/http"
	"time"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Meta    Meta        `json:"meta"`
}

type Meta struct {
	TraceID   string    `json:"trace_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func JSON(w http.ResponseWriter, status int, data interface{}, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: status >= 200 && status < 300,
		Data:    data,
		Meta: Meta{
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
		},
	})
}

func Error(w http.ResponseWriter, status int, message string, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: map[string]string{
			"message": message,
		},
		Meta: Meta{
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
		},
	})
}
