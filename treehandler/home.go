package treehandler

import (
	"context"
	"encoding/json"
	"net/http"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

type Tree4 struct {
	ID            string `json:"ID"`
	Name          string `json:"Name"`
	Published     bool   `json:"Published"`
	VolunteerName string `json:"volunteerName"` // ✅ Match JSON field
}

var app *firebase.App

func init() {
	ctx := context.Background()
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}
	var err error
	app, err = firebase.NewApp(ctx, conf, opt)
	if err != nil {
		panic("Failed to initialize Firebase App: " + err.Error())
	}
}

// Serve home.html
func HandleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/home.html")
}

// Count total tree records
func HandleTreeCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := context.Background()

	db, err := app.Database(ctx)
	if err != nil {
		http.Error(w, "DB init failed", http.StatusInternalServerError)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email parameter", http.StatusBadRequest)
		return
	}

	var trees map[string]Tree4
	err = db.NewRef("trees").Get(ctx, &trees)
	if err != nil {
		http.Error(w, "Failed to get tree data", http.StatusInternalServerError)
		return
	}

	count := 0
	for _, tree := range trees {
		if tree.VolunteerName == email && tree.Published {
			count++
		}
	}

	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
