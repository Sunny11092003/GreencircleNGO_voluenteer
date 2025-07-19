package treehandler

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

type Location4 struct {
	Address     string `json:"address"`
	City        string `json:"city"`
	Coordinates string `json:"coordinates"`
	Site        string `json:"site"`
}

type Classification3 struct {
	Class   string `json:"class"`
	Family  string `json:"family"`
	Genus   string `json:"genus"`
	Kingdom string `json:"kingdom"`
	Order   string `json:"order"`
	Phylum  string `json:"phylum"`
	Species string `json:"species"`
}

type ImageEntry1 struct {
	ImageType string `json:"imageType"`
	URL       string `json:"url"`
}

type TreeEntry10 struct {
	ID                    string          `json:"ID"`
	UID                   string          `json:"uid"`
	Timestamp             string          `json:"timestamp"`
	LastUpdated           string          `json:"lastUpdated"`
	Name                  string          `json:"Name"`
	Botanical             string          `json:"botanical"`
	Category              string          `json:"category"`
	Description           string          `json:"description"`
	MedicinalBenefits     string          `json:"medicinalBenefits"`
	EnvironmentalBenefits string          `json:"environmentalBenefits"`
	Native                string          `json:"native"`
	Published             bool            `json:"Published"`
	QR                    bool            `json:"QR"`
	Saved                 bool            `json:"Saved"`
	VolunteerName         string          `json:"volunteerName"`
	Location              Location4       `json:"location"`
	Classification        Classification3 `json:"classification"`
	ImagesRaw             json.RawMessage `json:"images"`
	Images                []ImageEntry1   `json:"-"`
}

func (t *TreeEntry10) ParseImages() error {
	var arr []ImageEntry1
	if err := json.Unmarshal(t.ImagesRaw, &arr); err == nil {
		t.Images = arr
		return nil
	}

	var obj map[string]ImageEntry1
	if err := json.Unmarshal(t.ImagesRaw, &obj); err != nil {
		return err
	}

	for _, val := range obj {
		t.Images = append(t.Images, val)
	}
	return nil
}

func ListTreesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	config := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}

	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		http.Error(w, "Firebase init failed", http.StatusInternalServerError)
		log.Println("App init error:", err)
		return
	}

	client, err := app.Database(ctx)
	if err != nil {
		http.Error(w, "Firebase DB connection failed", http.StatusInternalServerError)
		log.Println("DB init error:", err)
		return
	}

	// Get category
	category := r.URL.Query().Get("category")
	if category == "" {
		vars := mux.Vars(r)
		category = vars["category"]
	}
	if category == "" {
		http.Error(w, "Missing category", http.StatusBadRequest)
		return
	}

	// Determine view type (family or genera)
	viewType := "family"
	if strings.Contains(r.URL.Path, "/genera") {
		viewType = "genera"
	}

	// Fetch all trees
	var allTrees map[string]TreeEntry10
	err = client.NewRef("trees").Get(ctx, &allTrees)
	if err != nil {
		http.Error(w, "Error fetching trees", http.StatusInternalServerError)
		log.Println("Firebase read error:", err)
		return
	}

	// Deduplicate
	uniqueKeySet := make(map[string]bool)
	var uniqueTrees []TreeEntry10

	for _, t := range allTrees {
		if !t.Published {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(t.Category), strings.TrimSpace(category)) {
			var key string
			if viewType == "genera" {
				key = strings.TrimSpace(t.Classification.Genus)
			} else {
				key = strings.TrimSpace(t.Classification.Family)
			}
			if key == "" || uniqueKeySet[key] {
				continue
			}
			if err := t.ParseImages(); err != nil {
				log.Printf("Failed to parse images for tree %s: %v", t.ID, err)
				continue
			}
			uniqueKeySet[key] = true
			uniqueTrees = append(uniqueTrees, t)
		}
	}

	// Render template
	tmpl, err := template.ParseFiles("static/list.html")
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		log.Println("Template error:", err)
		return
	}

	err = tmpl.Execute(w, struct {
		Category string
		Trees    []TreeEntry10
		ViewType string
	}{
		Category: category,
		Trees:    uniqueTrees,
		ViewType: viewType,
	})
	if err != nil {
		log.Println("Template exec error:", err)
	}
}
