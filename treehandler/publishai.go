package treehandler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func PublishHandler1(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]

	if uid == "" {
		http.Error(w, "UID is required", http.StatusBadRequest)
		return
	}

	// Set published = true
	payload := map[string]bool{"Published": true}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
		return
	}

	firebaseURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s.json", uid)
	req, err := http.NewRequest("PATCH", firebaseURL, strings.NewReader(string(jsonData)))
	if err != nil {
		http.Error(w, "Failed to create PATCH request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Firebase PATCH failed:", err)
		http.Error(w, "Failed to update Firebase", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Println("Firebase response error:", string(bodyBytes))
		http.Error(w, "Firebase update failed", http.StatusInternalServerError)
		return
	}

	// Success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Tree %s marked as published", uid),
	})
}
