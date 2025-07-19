package treehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
)

const firebaseAPIKey = "AIzaSyDzlTwxJ161MSskkRAIfOA0GC3y3Wi4tME"
const firebaseAuthURL = "https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=" + firebaseAPIKey
const firebaseDBBaseURL = "https://login-credentials-b0464-default-rtdb.firebaseio.com/users" // no .json

type AuthSignupRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type AuthSignupResponse struct {
	IDToken string `json:"idToken"`
	Email   string `json:"email"`
	UID     string `json:"localId"`
}

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, filepath.Join("static", "signup.html"))
		return
	}

	// POST method: parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"Failed to parse form"}`, http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if email == "" || password == "" || confirm == "" {
		http.Error(w, `{"error":"All fields are required"}`, http.StatusBadRequest)
		return
	}

	if password != confirm {
		http.Error(w, `{"error":"Passwords do not match"}`, http.StatusBadRequest)
		return
	}

	// Firebase signup payload
	authPayload := AuthSignupRequest{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}

	payloadBytes, _ := json.Marshal(authPayload)

	// Send request to Firebase Auth
	resp, err := http.Post(firebaseAuthURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, `{"error":"Failed to contact Firebase"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Handle Firebase error response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": %s}`, string(body))))
		return
	}

	// Decode Firebase response
	var authResp AuthSignupResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		http.Error(w, `{"error":"Failed to parse Firebase response"}`, http.StatusInternalServerError)
		return
	}

	// Save user to Firebase Realtime DB
	userData := map[string]string{
		"email": authResp.Email,
		"uid":   authResp.UID,
	}
	userJSON, _ := json.Marshal(userData)

	dbURL := fmt.Sprintf("%s/%s.json", firebaseDBBaseURL, authResp.UID)
	req, err := http.NewRequest(http.MethodPut, dbURL, bytes.NewBuffer(userJSON))
	if err != nil {
		http.Error(w, `{"error":"Failed to create DB request"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	dbResp, err := client.Do(req)
	if err != nil || dbResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(dbResp.Body)
		http.Error(w, fmt.Sprintf(`{"error":"Database write failed: %s"}`, string(body)), http.StatusInternalServerError)
		return
	}
	defer dbResp.Body.Close()

	// ✅ Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
