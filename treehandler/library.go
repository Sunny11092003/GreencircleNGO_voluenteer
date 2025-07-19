package treehandler

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strings"
)

// TreeData4 holds the structure of a tree entry from Firebase
type TreeData4 struct {
	ID            string `json:"-"`
	Name          string `json:"Name"`
	Botanical     string `json:"botanical"`
	Timestamp     string `json:"timestamp"`
	Description   string `json:"description"`
	VolunteerName string `json:"volunteerName"`
	Location      struct {
		Site string `json:"site"`
	} `json:"location"`
	Images    []ImageData `json:"-"`
	QR        bool        `json:"QR"`
	Saved     bool        `json:"Saved"`
	Published bool        `json:"Published"`
}

type ImageData struct {
	URL string `json:"url"`
}

// Custom Unmarshal for mixed image formats
func (t *TreeData4) UnmarshalJSON(data []byte) error {
	// Create an alias to avoid infinite recursion
	type Alias TreeData4
	aux := &struct {
		Images json.RawMessage `json:"images"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try to unmarshal as array
	var arr []ImageData
	if err := json.Unmarshal(aux.Images, &arr); err == nil {
		t.Images = arr
		return nil
	}

	// Try to unmarshal as map
	var imgMap map[string]ImageData
	if err := json.Unmarshal(aux.Images, &imgMap); err != nil {
		return err
	}
	for _, img := range imgMap {
		t.Images = append(t.Images, img)
	}
	return nil
}

// ServelibraryHandlertree filters and serves tree entries belonging to a specific volunteer
func ServelibraryHandlertree(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email parameter", http.StatusBadRequest)
		return
	}

	resp, err := http.Get("https://treeqrsystem-default-rtdb.firebaseio.com/trees.json")
	if err != nil {
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Error decoding data", http.StatusInternalServerError)
		return
	}

	var filteredTrees []TreeData4
	for id, data := range raw {
		var tree TreeData4
		if err := json.Unmarshal(data, &tree); err != nil {
			continue
		}

		if strings.EqualFold(tree.VolunteerName, email) && tree.QR && tree.Saved && tree.Published {
			if idx := strings.Index(tree.Description, "."); idx != -1 {
				tree.Description = strings.TrimSpace(tree.Description[:idx+1])
			}
			tree.ID = id
			filteredTrees = append(filteredTrees, tree)
		}
	}

	// ✅ Safely parse the template
	tmpl, err := template.ParseFiles("static/library.html")
	if err != nil {
		http.Error(w, "Template parsing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ✅ Execute it safely
	if err := tmpl.Execute(w, filteredTrees); err != nil {
		http.Error(w, "Template rendering failed: "+err.Error(), http.StatusInternalServerError)
	}
}
