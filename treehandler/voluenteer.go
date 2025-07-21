// treehandler/handlers.go
package treehandler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

const firebaseDBURL1 = "https://login-credentials-b0464-default-rtdb.firebaseio.com/users.json"

func ServeVolunteersPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/voluenteers.html")
}

func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(firebaseDBURL1)
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func UpdateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		http.Error(w, "UID is required", http.StatusBadRequest)
		return
	}

	var payload struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	url := "https://login-credentials-b0464-default-rtdb.firebaseio.com/users/" + uid + "/role.json"

	jsonBody, _ := json.Marshal(payload.Role)
	req, err := http.NewRequest(http.MethodPut, url, io.NopCloser(bytes.NewReader(jsonBody)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to update role", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Role updated successfully"))
}
