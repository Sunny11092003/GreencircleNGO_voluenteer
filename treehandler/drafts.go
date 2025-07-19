package treehandler

import (
	"context"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type TreeData3 struct {
	ID            string `json:"ID"`
	UID           string `json:"uid"`
	Name          string `json:"Name"`
	Timestamp     string `json:"timestamp"`
	QR            bool   `json:"QR"`
	Saved         bool   `json:"Saved"`
	Published     bool   `json:"Published"`
	VolunteerName string `json:"volunteerName"`
}

func ServeQRHandlertreedrafts(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email parameter", http.StatusBadRequest)
		return
	}

	resp, err := http.Get("https://treeqrsystem-default-rtdb.firebaseio.com/trees.json")
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read data", http.StatusInternalServerError)
		return
	}

	var result map[string]TreeData3
	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, "Failed to parse data", http.StatusInternalServerError)
		return
	}

	var filteredTrees []TreeData3
	for _, tree := range result {
		if !tree.Published && tree.QR && strings.EqualFold(tree.VolunteerName, email) {
			filteredTrees = append(filteredTrees, tree)
		}
	}

	tmpl := template.Must(template.ParseFiles("static/drafts.html"))
	if err := tmpl.Execute(w, filteredTrees); err != nil {
		http.Error(w, "Template rendering failed", http.StatusInternalServerError)
	}
}

// DeleteTreeHandlerdrafts deletes a tree from Firebase Realtime Database
func DeleteTreeHandlerdrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	firebaseKey := r.FormValue("uid")
	email := r.FormValue("email")

	if firebaseKey == "" {
		http.Error(w, "Missing Firebase UID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}
	app, err := firebase.NewApp(ctx, conf, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Fatalln("Error initializing app:", err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}

	// 🔥 Delete directly using the Firebase key (uid)
	ref := client.NewRef("trees/" + firebaseKey)
	if err := ref.Delete(ctx); err != nil {
		http.Error(w, "Failed to delete record", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted Firebase UID:", firebaseKey)
	http.Redirect(w, r, "/drafts?email="+url.QueryEscape(email), http.StatusSeeOther)
}
