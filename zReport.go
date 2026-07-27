package main

import (
	"io"
	"net/http"
)

func ReportGdps(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	data := string(body)

	TGWebhookLog(data)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
