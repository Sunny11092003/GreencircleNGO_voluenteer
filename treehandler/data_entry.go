package treehandler

import (
	"context"
	"html/template"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type TreeData1 struct {
	Name                  string `json:"name"`
	Botanical             string `json:"botanical"`
	Description           string `json:"description"`
	Category              string `json:"category"`
	Native                string `json:"native"`
	MedicinalBenefits     string `json:"medicinalBenefits"`
	EnvironmentalBenefits string `json:"environmentalBenefits"`
}

func DataEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing tree ID", http.StatusBadRequest)
		return
	}

	// Firebase init
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
		http.Error(w, "Firebase DB client error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ref := client.NewRef("trees/" + id)

	switch r.Method {
	case http.MethodGet:
		var existingData map[string]interface{}
		if err := ref.Get(ctx, &existingData); err != nil {
			http.Error(w, "Failed to fetch data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("static/data_entry.html")
		if err != nil {
			http.Error(w, "Template parse error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]interface{}{
			"TreeID": id,
			"Name":   existingData["name"],
		})
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Form parse error", http.StatusBadRequest)
			return
		}

		data := TreeData1{
			Name:                  r.FormValue("name"),
			Botanical:             r.FormValue("botinical"),
			Description:           r.FormValue("description"),
			Category:              r.FormValue("category"),
			Native:                r.FormValue("native"),
			MedicinalBenefits:     r.FormValue("medbenefits"),
			EnvironmentalBenefits: r.FormValue("envibenefits"),
		}

		updateData := map[string]interface{}{
			"Name":                  data.Name,
			"botanical":             data.Botanical,
			"description":           data.Description,
			"category":              data.Category,
			"native":                data.Native,
			"medicinalBenefits":     data.MedicinalBenefits,
			"environmentalBenefits": data.EnvironmentalBenefits,
			"Saved":                 true, // Set the Saved flag to true
			"lastUpdated":           time.Now().Format("2006-01-02 15:04:05"),
		}

		if err := ref.Update(ctx, updateData); err != nil {
			http.Error(w, "Failed to update Firebase: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/classification?id="+id, http.StatusSeeOther)

	}

}
