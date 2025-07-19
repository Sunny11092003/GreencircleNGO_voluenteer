package treehandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Location1 struct {
	Coordinates string `json:"coordinates"`
	Site        string `json:"site"`
	Address     string `json:"address"`
	City        string `json:"city"`
}

type SaveData struct {
	ResponseText string    `json:"responseText"`
	LastUpdated  string    `json:"lastUpdated"`
	Location     Location1 `json:"location"`
}

type FinalData struct {
	Description           string            `json:"description"`
	MedicinalBenefits     string            `json:"medicinalBenefits"`
	EnvironmentalBenefits string            `json:"environmentalBenefits"`
	Native                string            `json:"native"`
	Classification        map[string]string `json:"classification"`
	LastUpdated           string            `json:"lastUpdated"`
	Location              Location1         `json:"location"`
	Category              string            `json:"category"` // ← updated field name
}

func SaveAIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]

	var incoming SaveData
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Split responseText into fields
	sections, classification := parseResponse(incoming.ResponseText)

	// Construct final payload
	payload := FinalData{
		Description:           sections["description"],
		MedicinalBenefits:     sections["medicinalBenefits"],
		EnvironmentalBenefits: sections["environmentalBenefits"],
		Native:                sections["native"],
		Classification:        classification,
		LastUpdated:           time.Now().Format("2006-01-02 15:04:05"),
		Location:              incoming.Location,
		Category:              strings.TrimSpace(sections["category"]), // ← final field
	}

	// Marshal and PATCH to Firebase
	jsonData, _ := json.Marshal(payload)
	firebaseURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s.json", uid) // <-- Updated line
	req, _ := http.NewRequest("PATCH", firebaseURL, strings.NewReader(string(jsonData)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to save to Firebase", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully saved"})

}

func parseResponse(text string) (map[string]string, map[string]string) {
	sections := map[string]string{
		"description":           "",
		"medicinalBenefits":     "",
		"environmentalBenefits": "",
		"native":                "",
		"category":              "",
	}

	classification := map[string]string{}
	lines := strings.Split(text, "\n")
	current := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Detailed Description:"):
			current = "description"
			sections[current] = strings.TrimPrefix(line, "Detailed Description:")
		case strings.HasPrefix(line, "Medicinal Benefits:"):
			current = "medicinalBenefits"
			sections[current] = strings.TrimPrefix(line, "Medicinal Benefits:")
		case strings.HasPrefix(line, "Environmental Benefits:"):
			current = "environmentalBenefits"
			sections[current] = strings.TrimPrefix(line, "Environmental Benefits:")
		case strings.HasPrefix(line, "Native to India:"):
			current = "native"
			nativeText := strings.TrimSpace(strings.TrimPrefix(line, "Native to India:"))
			if strings.HasPrefix(strings.ToLower(nativeText), "yes") {
				sections["native"] = "Yes"
			} else {
				sections["native"] = "No"
			}
		case strings.HasPrefix(line, "Scientific Classification:"):
			current = "classification"
		case strings.HasPrefix(line, "Common Tree Category:"):
			current = "category"
			sections[current] = strings.TrimPrefix(line, "Common Tree Category:")
		default:
			if current == "classification" && strings.HasPrefix(line, "- ") {
				parts := strings.SplitN(strings.TrimPrefix(line, "- "), ":", 2)
				if len(parts) == 2 {
					key := strings.ToLower(strings.TrimSpace(parts[0]))
					value := strings.TrimSpace(parts[1])
					classification[key] = value
				}
			} else if current != "" {
				sections[current] += " " + line
			}
		}
	}

	for k, v := range sections {
		sections[k] = strings.TrimSpace(v)
	}
	return sections, classification
}
