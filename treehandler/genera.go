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

func (t *TreeEntry10) ParseImages1() error {
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

func ListGeneraTreesHandler(w http.ResponseWriter, r *http.Request) {
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

	category := r.URL.Query().Get("category")
	if category == "" {
		vars := mux.Vars(r)
		category = vars["category"]
	}
	if category == "" {
		http.Error(w, "Missing category", http.StatusBadRequest)
		return
	}

	var allTrees map[string]TreeEntry10
	err = client.NewRef("trees").Get(ctx, &allTrees)
	if err != nil {
		http.Error(w, "Error fetching trees", http.StatusInternalServerError)
		log.Println("Firebase read error:", err)
		return
	}

	genusSet := make(map[string]bool)
	var uniqueTrees []TreeEntry10

	for _, t := range allTrees {
		if !t.Published {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(t.Category), strings.TrimSpace(category)) {
			genus := strings.TrimSpace(t.Classification.Genus)
			if genus == "" || genusSet[genus] {
				continue
			}
			if err := t.ParseImages1(); err != nil {
				log.Printf("Failed to parse images for tree %s: %v", t.ID, err)
				continue
			}
			genusSet[genus] = true
			uniqueTrees = append(uniqueTrees, t)
		}
	}

	// ✅ Use genera.html here
	tmpl, err := template.ParseFiles("static/genera.html")
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		log.Println("Template error:", err)
		return
	}

	err = tmpl.Execute(w, struct {
		Category string
		Trees    []TreeEntry10
	}{
		Category: category,
		Trees:    uniqueTrees,
	})
	if err != nil {
		log.Println("Template exec error:", err)
	}
}
