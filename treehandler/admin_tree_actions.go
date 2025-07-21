package treehandler

import (
	"context"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

// ✏️ EDIT Tree Handler (display page for now)

// 🗑 DELETE Tree Handler
func DeleteTreeHandleradmin(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		http.Error(w, "Firebase init failed", http.StatusInternalServerError)
		log.Println("firebase.NewApp:", err)
		return
	}

	db, err := app.Database(ctx)
	if err != nil {
		http.Error(w, "Firebase DB failed", http.StatusInternalServerError)
		log.Println("app.Database:", err)
		return
	}

	if err := db.NewRef("trees/" + id).Delete(ctx); err != nil {
		http.Error(w, "Failed to delete tree", http.StatusInternalServerError)
		log.Println("delete error:", err)
		return
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}
