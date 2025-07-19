package treehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"google.golang.org/api/option"
)

// Initialize Firebase with Realtime DB URL
func initializeFirebaseApp() *firebase.App {
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	config := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/", // ✅ replace if different
	}

	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		log.Printf("🔥 Firebase init error: %v", err)
		return nil
	}
	return app
}

// Handler: POST /delete-image
// Payload: { "uid": "<uid>", "url": "<cloudinary-url>" }
func DeleteImageHandler(w http.ResponseWriter, r *http.Request) {
	type req struct {
		UID string `json:"uid"`
		URL string `json:"url"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad JSON", http.StatusBadRequest)
		return
	}
	log.Printf("📥 Delete request: UID=%s, URL=%s", body.UID, body.URL)

	// --- Initialize Firebase ---
	app := initializeFirebaseApp()
	if app == nil {
		http.Error(w, "failed to initialize Firebase app", http.StatusInternalServerError)
		return
	}

	db, err := app.Database(context.Background())
	if err != nil || db == nil {
		log.Printf("🔥 Failed to get DB client: %v", err)
		http.Error(w, "Firebase DB error", http.StatusInternalServerError)
		return
	}

	// --- Read from Firebase Realtime DB ---
	ref := db.NewRef(fmt.Sprintf("trees/%s/images", body.UID))
	var imgs []map[string]string

	if err := ref.Get(context.Background(), &imgs); err != nil {
		log.Printf("⚠️ Failed to fetch images: %v", err)
		http.Error(w, "Could not fetch images", http.StatusInternalServerError)
		return
	}

	log.Printf("📦 Found %d image(s)", len(imgs))

	// --- Filter out the image to delete ---
	filter := make([]map[string]string, 0, len(imgs))
	found := false

	for _, m := range imgs {
		log.Printf("🔍 Checking image: %s", m["url"])
		if m["url"] == body.URL {
			log.Printf("✅ Match found — deleting this image")
			found = true
			continue
		}
		filter = append(filter, m)
	}

	if !found {
		log.Printf("❌ No matching image found to delete.")
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// --- Update Firebase with filtered image list ---
	if err := ref.Set(context.Background(), filter); err != nil {
		log.Printf("🔥 Failed to update image list: %v", err)
		http.Error(w, "Could not update image list", http.StatusInternalServerError)
		return
	}
	log.Printf("✅ Firebase DB updated — image removed from UID %s", body.UID)

	// --- Delete from Cloudinary ---
	publicID := cloudPublicID(body.URL)
	log.Printf("☁️ Cloudinary public ID: %s", publicID)

	cld, err := cloudinary.NewFromParams("dybnayjf6", "676827133671535", "ZM8oGKcKNsXzaTjHZ27GRGRQ7CE")
	if err != nil {
		log.Printf("☁️ Cloudinary init failed: %v", err)
		http.Error(w, "Cloudinary init error", http.StatusInternalServerError)
		return
	}

	_, err = cld.Upload.Destroy(context.TODO(), uploader.DestroyParams{PublicID: publicID})
	if err != nil {
		log.Printf("❌ Cloudinary delete failed: %v", err)
		http.Error(w, "Cloudinary delete error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Cloudinary image deleted: %s", body.URL)

	// --- Final response ---
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": body.URL,
	})
}

// Extracts Cloudinary public ID from the full image URL
func cloudPublicID(url string) string {
	parts := strings.Split(url, "/upload/")
	if len(parts) < 2 {
		return ""
	}
	// e.g., upload/treeqr/uid/image.jpg → treeqr/uid/image
	return strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
}
