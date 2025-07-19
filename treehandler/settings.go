package treehandler

import (
	"html/template"
	"net/http"
)

// RenderSettingPage serves the settings page
func RenderSettingPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("static/settings.html")
	if err != nil {
		http.Error(w, "Failed to load settings page", http.StatusInternalServerError)
		return
	}

	// Optional: Pass data if needed
	tmpl.Execute(w, nil)
}
