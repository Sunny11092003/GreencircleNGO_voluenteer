package treehandler

import (
	"context"
	"html/template"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type ClassificationData struct {
	Kingdom string `json:"kingdom"`
	Phylum  string `json:"phylum"`
	Class   string `json:"class"`
	Order   string `json:"order"`
	Family  string `json:"family"`
	Genus   string `json:"genus"`
	Species string `json:"species"`
}

func ClassificationHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing tree ID", http.StatusBadRequest)
			return
		}

		tmpl, err := template.ParseFiles("static/classification.html")
		if err != nil {
			http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]interface{}{"TreeID": id})

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Form parse error", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "Missing tree ID", http.StatusBadRequest)
			return
		}

		data := ClassificationData{
			Kingdom: r.FormValue("kingdom"),
			Phylum:  r.FormValue("phylum"),
			Class:   r.FormValue("class"),
			Order:   r.FormValue("order"),
			Family:  r.FormValue("family"),
			Genus:   r.FormValue("genus"),
			Species: r.FormValue("species"),
		}

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

		ref := client.NewRef("trees/" + id + "/classification")
		if err := ref.Set(ctx, data); err != nil {
			http.Error(w, "Failed to save classification data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// ✅ Redirect to /images?id=<id>
		http.Redirect(w, r, "/location?id="+id, http.StatusSeeOther)
	}
}
