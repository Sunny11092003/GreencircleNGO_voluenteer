package treehandler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

// ---------- Structs ----------

type Location5 struct {
	Address     string `json:"address"`
	City        string `json:"city"`
	Coordinates string `json:"coordinates"`
	Site        string `json:"site"`
}

type Classification2 struct {
	Class   string `json:"class"`
	Family  string `json:"family"`
	Genus   string `json:"genus"`
	Kingdom string `json:"kingdom"`
	Order   string `json:"order"`
	Phylum  string `json:"phylum"`
	Species string `json:"species"`
}

type ImageEntry struct {
	ImageType string `json:"imageType"`
	URL       string `json:"url"`
}

type TreeEntry struct {
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
	Location              Location5       `json:"location"`
	Classification        Classification2 `json:"classification"`
	ImagesRaw             json.RawMessage `json:"images"` // Raw format for decoding either array or map
	Images                []ImageEntry    `json:"-"`      // Final parsed images
}

// ---------- Image Normalization ----------

func (t *TreeEntry) NormalizeImages() error {
	// Try to decode as array first
	var arr []ImageEntry
	if err := json.Unmarshal(t.ImagesRaw, &arr); err == nil {
		t.Images = arr
		return nil
	}

	// Try to decode as map
	var m map[string]ImageEntry
	if err := json.Unmarshal(t.ImagesRaw, &m); err == nil {
		for _, v := range m {
			t.Images = append(t.Images, v)
		}
		return nil
	}

	return fmt.Errorf("unsupported image format")
}

// ---------- HTTP Handler ----------

func GetEventHandler(w http.ResponseWriter, r *http.Request) {
	// Get event ID from query parameter
	uid := r.URL.Query().Get("id")
	if uid == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Fetch data from Firebase
	resp, err := http.Get("https://treeqrsystem-default-rtdb.firebaseio.com/trees.json")
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Decode all tree entries
	all := make(map[string]*TreeEntry)
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		http.Error(w, "Failed to decode data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Find the tree with matching UID
	entry, ok := all[uid]
	if !ok {
		http.Error(w, "Tree not found", http.StatusNotFound)
		return
	}

	entry.UID = uid // Assign UID

	// Normalize image field
	if err := entry.NormalizeImages(); err != nil {
		http.Error(w, "Failed to normalize images: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load and render the HTML template
	tmpl := template.Must(template.ParseFiles("static/fetch.html"))
	if err := tmpl.Execute(w, entry); err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}
