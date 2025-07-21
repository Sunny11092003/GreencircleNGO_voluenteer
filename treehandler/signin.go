package treehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

const firebaseLoginURL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=AIzaSyDzlTwxJ161MSskkRAIfOA0GC3y3Wi4tME"
const firebaseDBURL = "https://login-credentials-b0464-default-rtdb.firebaseio.com/users" // Replace with your actual DB URL

type AuthLoginRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type AuthLoginResponse struct {
	IDToken string `json:"idToken"`
	Email   string `json:"email"`
	UID     string `json:"localId"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, filepath.Join("static", "sign.html"))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	loginPayload := AuthLoginRequest{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}

	payloadBytes, _ := json.Marshal(loginPayload)

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

	var loginResp AuthLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		http.Error(w, "Failed to parse login response", http.StatusInternalServerError)
		return
	}

	// 🔐 Check verification status
	userDataURL := fmt.Sprintf("%s/%s.json", firebaseDBURL, loginResp.UID)
	userResp, err := http.Get(userDataURL)
	if err != nil {
		http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	body, _ := ioutil.ReadAll(userResp.Body)
	var userData map[string]interface{}
	if err := json.Unmarshal(body, &userData); err != nil {
		http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
		return
	}

	verified, ok := userData["verified"].(bool)
	if !ok || !verified {
		http.Error(w, "Account not verified", http.StatusForbidden)
		return
	}

	// ✅ Success
	w.Write([]byte("success"))
}
