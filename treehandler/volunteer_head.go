package treehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

const (
	firebaseURL    = "https://login-credentials-b0464-default-rtdb.firebaseio.com/"
	serviceAccount = "login-credentials-b0464-firebase-adminsdk-fbsvc-627ab92f3c.json"
)

/* ──────────────── FIREBASE ──────────────── */

func firebaseApp() (*firebase.App, error) {
	return firebase.NewApp(context.Background(), &firebase.Config{
		DatabaseURL: firebaseURL,
	}, option.WithCredentialsFile(serviceAccount))
}

func dbClient() (*db.Client, error) {
	app, err := firebaseApp()
	if err != nil {
		return nil, err
	}
	return app.Database(context.Background())
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

/* ──────────────── PAGES ──────────────── */

func HeadDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("static", "head_dashboard.html"))
}

/* ──────────────── PENDING LIST ──────────────── */

func HeadPending(w http.ResponseWriter, _ *http.Request) {
	db, err := dbClient()
	if err != nil {
		fmt.Println("🔥 Error initializing Firebase DB client:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var users map[string]map[string]interface{}
	err = db.NewRef("users").Get(context.Background(), &users)
	if err != nil {
		fmt.Println("🔥 Error fetching users from Firebase:", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	fmt.Printf("✅ Successfully fetched %d users\n", len(users))

	type row struct {
		Email     string `json:"email"`
		Timestamp string `json:"timestamp,omitempty"`
	}

	var out []row
	for _, v := range users {
		verified, ok := v["verified"].(bool)
		if ok && !verified {
			out = append(out, row{
				Email:     toString(v["email"]),
				Timestamp: toString(v["timestamp"]),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

/* ──────────────── APPROVE VOLUNTEER ──────────────── */

func HeadApprove(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Email      string `json:"email"`
		ApprovedBy string `json:"approvedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	db, err := dbClient()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var user map[string]interface{}
	if err := db.NewRef("users").Get(context.Background(), &user); err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	// Find UID with matching email
	for uid, u := range user {
		if toString(u.(map[string]interface{})["email"]) == p.Email {
			now := time.Now().Format(time.RFC3339)
			updates := map[string]interface{}{
				"verified":   true,
				"approvedBy": p.ApprovedBy,
				"approvedAt": now,
			}
			if err := db.NewRef("users/"+uid).Update(context.Background(), updates); err != nil {
				http.Error(w, "Failed to approve volunteer", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "approved",
				"email":  p.Email,
			})
			return
		}
	}

	http.Error(w, "Volunteer not found", http.StatusNotFound)
}

/* ──────────────── REJECT VOLUNTEER ──────────────── */

func HeadReject(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	db, err := dbClient()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var user map[string]interface{}
	if err := db.NewRef("users").Get(context.Background(), &user); err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	for uid, u := range user {
		if toString(u.(map[string]interface{})["email"]) == p.Email {
			if err := db.NewRef("users/" + uid).Delete(context.Background()); err != nil {
				http.Error(w, "Failed to reject volunteer", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "rejected",
				"email":  p.Email,
			})
			return
		}
	}

	http.Error(w, "Volunteer not found", http.StatusNotFound)
}

/* ──────────────── VERIFIED LIST ──────────────── */

func GetVerifiedVolunteers(c *gin.Context) {
	db, err := dbClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var users map[string]map[string]interface{}
	if err := db.NewRef("users").Get(context.Background(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	type Volunteer struct {
		Email      string `json:"email"`
		ApprovedBy string `json:"approved_by,omitempty"`
		ApprovedAt string `json:"approved_at,omitempty"`
	}

	var volunteers []Volunteer
	for _, u := range users {
		if verified, ok := u["verified"].(bool); ok && verified {
			volunteers = append(volunteers, Volunteer{
				Email:      toString(u["email"]),
				ApprovedBy: toString(u["approvedBy"]),
				ApprovedAt: toString(u["approvedAt"]),
			})
		}
	}

	c.JSON(http.StatusOK, volunteers)
}
