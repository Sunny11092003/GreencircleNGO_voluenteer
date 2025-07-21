package treehandler

import (
	"context"
	"html/template"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

/* ------------------- form data sent to the template ------------------- */
type EditTreeForm struct {
	ID          string
	Name        string
	Botanical   string
	Site        string
	Volunteer   string
	Published   bool
	QR          bool
	Description string
	Category    string
	Medicinal   string
}

/* -------------------------- EditTreeHandler --------------------------- */
func EditTreeHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	log.Println("✏️ EditTreeHandler called, id =", id)
	if id == "" {
		http.NotFound(w, r)
		return
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Println("❌ firebase.NewApp:", err)
		http.Error(w, "Firebase init failed", http.StatusInternalServerError)
		return
	}
	db, err := app.Database(ctx)
	if err != nil {
		log.Println("❌ app.Database:", err)
		http.Error(w, "DB connection error", http.StatusInternalServerError)
		return
	}

	// 📝 POST: Save changes
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad form data", http.StatusBadRequest)
			return
		}

		update := map[string]interface{}{
			"Name":              r.FormValue("name"),
			"botanical":         r.FormValue("botanical"),
			"category":          r.FormValue("category"),
			"description":       r.FormValue("description"),
			"medicinalBenefits": r.FormValue("medicinal"),
			"location/site":     r.FormValue("site"),
		}

		if err := db.NewRef("trees/"+id).Update(ctx, update); err != nil {
			log.Println("❌ update error:", err)
			http.Error(w, "DB update error", http.StatusInternalServerError)
			return
		}

		log.Println("✅ tree updated:", id)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	// 📄 GET: Load form
	var tree map[string]interface{}
	if err := db.NewRef("trees/"+id).Get(ctx, &tree); err != nil {
		log.Println("❌ fetch tree:", err)
		http.NotFound(w, r)
		return
	}

	form := EditTreeForm{
		ID:          id,
		Name:        toString(tree["Name"]),
		Botanical:   toString(tree["botanical"]),
		Category:    toString(tree["category"]),
		Volunteer:   toString(tree["volunteerName"]),
		Published:   toBool(tree["Published"]),
		QR:          toBool(tree["QR"]),
		Description: toString(tree["description"]),
		Medicinal:   toString(tree["medicinalBenefits"]),
	}

	if loc, ok := tree["location"].(map[string]interface{}); ok {
		form.Site = toString(loc["site"])
	}

	tmpl, err := template.ParseFiles("static/edit_tree.html")
	if err != nil {
		log.Println("❌ template parse:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, form); err != nil {
		log.Println("❌ template execute:", err)
	}
}
