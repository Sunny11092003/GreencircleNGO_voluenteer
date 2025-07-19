package treehandler

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// Structs for parsing JSON data
type Classification struct {
	Class   string `json:"class"`
	Family  string `json:"family"`
	Genus   string `json:"genus"`
	Kingdom string `json:"kingdom"`
	Order   string `json:"order"`
	Phylum  string `json:"phylum"`
	Species string `json:"species"`
}

type Location struct {
	Address     string `json:"address"`
	City        string `json:"city"`
	Coordinates string `json:"coordinates"`
	Site        string `json:"site"`
}

type Image struct {
	ImageType string `json:"imageType"`
	URL       string `json:"url"`
}

type TreeData2 struct {
	UID                   string         `json:"uid"`
	ID                    string         `json:"ID"`
	Name                  string         `json:"Name"`
	Botanical             string         `json:"botanical"`
	Category              string         `json:"category"`
	Description           string         `json:"description"`
	MedicinalBenefits     string         `json:"medicinalBenefits"`
	EnvironmentalBenefits string         `json:"environmentalBenefits"`
	Classification        Classification `json:"classification"`
	Location              Location       `json:"location"`
	Images                []Image        `json:"images"`
	Published             bool           `json:"Published"` // ✅ Add this
}

func CompleteHandler(w http.ResponseWriter, r *http.Request) {
	treeID := r.URL.Query().Get("id")
	if treeID == "" {
		http.Error(w, "Missing tree ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}

	app, err := firebase.NewApp(ctx, conf, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v", err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("Error getting database client: %v", err)
	}

	var tree TreeData2
	ref := client.NewRef("trees/" + treeID)
	if err := ref.Get(ctx, &tree); err != nil {
		log.Printf("Error fetching tree data: %v", err)
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	// Clean up values
	tree.MedicinalBenefits = strings.TrimSpace(tree.MedicinalBenefits)
	tree.EnvironmentalBenefits = strings.TrimSpace(tree.EnvironmentalBenefits)

	tmpl, err := template.ParseFiles("static/complete.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, tree); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}
