package treehandler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

const firebaseAPIKey1 = "AIzaSyDzlTwxJ161MSskkRAIfOA0GC3y3Wi4tME"
const loginURL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=" + firebaseAPIKey1
const updatePasswordURL = "https://identitytoolkit.googleapis.com/v1/accounts:update?key=" + firebaseAPIKey1

type PasswordChangeRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type loginResponse struct {
	IDToken string `json:"idToken"`
}

type passwordUpdateRequest struct {
	IDToken           string `json:"idToken"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	// Step 1: Login to get ID Token
	loginPayload := map[string]interface{}{
		"email":             req.Email,
		"password":          req.CurrentPassword,
		"returnSecureToken": true,
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginResp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(loginBody))
	if err != nil || loginResp.StatusCode != 200 {
		http.Error(w, `{"error":"Invalid current password"}`, http.StatusUnauthorized)
		return
	}
	var login loginResponse
	body, _ := io.ReadAll(loginResp.Body)
	json.Unmarshal(body, &login)

	// Step 2: Change password
	updatePayload := passwordUpdateRequest{
		IDToken:           login.IDToken,
		Password:          req.NewPassword,
		ReturnSecureToken: true,
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateResp, err := http.Post(updatePasswordURL, "application/json", bytes.NewBuffer(updateBody))
	if err != nil || updateResp.StatusCode != 200 {
		http.Error(w, `{"error":"Password update failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"Password changed successfully"}`))
}
