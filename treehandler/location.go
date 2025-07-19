package treehandler

import (
	"context"
	"html/template"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type LocationData struct {
	Coordinates string `json:"coordinates"`
	Site        string `json:"site"`
	Address     string `json:"address"`
	City        string `json:"city"`
}

func LocationHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		// Extract tree ID from URL
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing tree ID", http.StatusBadRequest)
			return
		}

		// Load and render the HTML template
		tmpl, err := template.ParseFiles("static/location.html")
		if err != nil {
			http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Pass TreeID to the template
		tmpl.Execute(w, map[string]interface{}{"TreeID": id})

	case http.MethodPost:
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Form parse error", http.StatusBadRequest)
			return
		}

		// Get tree ID and form fields
		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "Missing tree ID", http.StatusBadRequest)
			return
		}

		data := LocationData{
			Coordinates: r.FormValue("coordinates"),
			Site:        r.FormValue("site"),
			Address:     r.FormValue("address"),
			City:        r.FormValue("city"),
		}

		// Initialize Firebase
		ctx := context.Background()
		app, err := firebase.NewApp(ctx, &firebase.Config{
			DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
		}, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
		if err != nil {
			http.Error(w, "Firebase init failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		client, err := app.Database(ctx)
		if err != nil {
			http.Error(w, "Database client error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Save to Firebase under /trees/<id>/location
		ref := client.NewRef("trees/" + id + "/location")
		if err := ref.Set(ctx, data); err != nil {
			http.Error(w, "Failed to save location data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to the next step
		http.Redirect(w, r, "/image?id="+id, http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
