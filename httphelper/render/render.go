package render

import (
	"encoding/json"
	"net/http"
	"time"
)

type PaginationResponse struct {
	Pagination struct {
		Current    uint64 `json:"current"`
		Offset     uint64 `json:"offset"`
		Limit      uint64 `json:"limit"`
		NextOffset uint64 `json:"next_offset"`
		Total      uint64 `json:"total"`
	} `json:"pagination"`
	Langs []string `json:"langs"`
	Items any      `json:"items"`
}

func Html(w http.ResponseWriter, t []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(t)
}

func Text(w http.ResponseWriter, t []byte) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(t)
}

func NotFound(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(err.Error()))
}

func Error(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	msg := "Unknown"
	if err != nil {
		msg = err.Error()
	}
	_ = enc.Encode(map[string]interface{}{
		"msg": msg,
	})
}

func JSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	_ = enc.Encode(map[string]any{
		"ts":   time.Now().Unix(),
		"data": v,
	})
}

func JSONRaw(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func JSONBytes(w http.ResponseWriter, bs []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bs)
}
