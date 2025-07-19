package treehandler

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func PublishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	treeUID := r.FormValue("uid")
	if treeUID == "" {
		http.Error(w, "Missing UID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	conf := &firebase.Config{DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/"}
	app, err := firebase.NewApp(ctx, conf, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Printf("Firebase init error: %v", err)
		http.Error(w, "Failed to initialize Firebase", http.StatusInternalServerError)
		return
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Printf("Database init error: %v", err)
		http.Error(w, "Failed to connect to Firebase DB", http.StatusInternalServerError)
		return
	}

	ref := client.NewRef("trees/" + treeUID)

	// ✅ Fetch existing tree data
	var treeData map[string]interface{}
	if err := ref.Get(ctx, &treeData); err != nil {
		log.Printf("Failed to get tree data: %v", err)
		http.Error(w, "Failed to read tree data", http.StatusInternalServerError)
		return
	}

	// ✅ If "ID" is missing, generate from "Name"
	if _, ok := treeData["ID"]; !ok {
		nameVal, nameExists := treeData["Name"].(string)
		if !nameExists || nameVal == "" {
			http.Error(w, "Tree Name is required to generate ID", http.StatusBadRequest)
			return
		}

		newID := generateIDFromName(nameVal)

		err := ref.Update(ctx, map[string]interface{}{
			"ID": newID,
		})
		if err != nil {
			log.Printf("Failed to update ID: %v", err)
			http.Error(w, "Failed to assign generated ID", http.StatusInternalServerError)
			return
		}
	}

	// ✅ Set Published = true
	err = ref.Update(ctx, map[string]interface{}{
		"Published": true,
	})
	if err != nil {
		log.Printf("Update error: %v", err)
		http.Error(w, "Failed to publish the tree", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

// ✅ Generate ID like: appleblossom-shower-1125
func generateIDFromName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-%d", name, rand.Intn(9000)+1000) // ensures 4-digit suffix
}
