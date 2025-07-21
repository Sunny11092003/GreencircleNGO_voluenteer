package treehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

func LoginHandleradmin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, filepath.Join("static", "admin_login.html"))
		return
	}

	// Parse JSON body
	var creds AuthLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if creds.Email == "" || creds.Password == "" {
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	creds.ReturnSecureToken = true
	payloadBytes, _ := json.Marshal(creds)

	resp, err := http.Post(firebaseLoginURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Firebase connection failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	var authResp AuthLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		http.Error(w, "Failed to decode Firebase response", http.StatusInternalServerError)
		return
	}

	userDataURL := fmt.Sprintf("%s/users/%s.json", databaseURL, authResp.UID)
	userResp, err := http.Get(userDataURL)
	if err != nil || userResp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(userResp.Body)

	var user FirebaseUser
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Unauthorized role", http.StatusForbidden)
		return
	}

	// ✅ All good
	w.Write([]byte("success"))
}
