package treehandler

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"google.golang.org/api/option"
)

type ImageMeta struct {
	URL       string `json:"url"`
	ImageType string `json:"imageType"`
}

func UploadImagesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	treeID := r.URL.Query().Get("id")
	if treeID == "" {
		http.Error(w, "Missing ID in URL", http.StatusBadRequest)
		return
	}

	// Firebase init
	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}
	app, err := firebase.NewApp(ctx, conf, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		http.Error(w, "Firebase initialization error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	client, err := app.Database(ctx)
	if err != nil {
		http.Error(w, "Firebase DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// For GET: serve the HTML upload page with image count & thumbnail
	if r.Method == http.MethodGet {
		ref := client.NewRef("trees/" + treeID + "/images")
		var existing []ImageMeta
		_ = ref.Get(ctx, &existing) // ignore error; assume 0 if not found

		var thumb string
		if len(existing) > 0 {
			rand.Seed(time.Now().UnixNano())
			thumb = existing[rand.Intn(len(existing))].URL
		}

		tmpl, err := template.ParseFiles("static/images.html")
		if err != nil {
			http.Error(w, "Unable to load template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]interface{}{
			"ID":    treeID,
			"Count": len(existing),
			"Thumb": thumb,
		})
		return
	}

	// Reject non-POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err = r.ParseMultipartForm(20 << 20)
	if err != nil {
		http.Error(w, "Error parsing form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	imageType := r.FormValue("imageType")
	if imageType == "" {
		writeAlert(w, "Missing image type")
		return
	}

	// Cloudinary init
	cld, err := cloudinary.NewFromParams("dybnayjf6", "676827133671535", "ZM8oGKcKNsXzaTjHZ27GRGRQ7CE")
	if err != nil {
		writeAlert(w, "Cloudinary init error: "+err.Error())
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		writeAlert(w, "No images selected")
		return
	}

	// Fetch existing images
	ref := client.NewRef("trees/" + treeID + "/images")
	var existing []ImageMeta
	err = ref.Get(ctx, &existing)
	if err != nil {
		log.Println("Warning: Could not fetch existing images:", err)
		existing = []ImageMeta{}
	}

	remainingSlots := 4 - len(existing)
	if remainingSlots <= 0 {
		writeAlert(w, "Maximum of 4 images already uploaded")
		return
	}
	if len(files) > remainingSlots {
		writeAlert(w, fmt.Sprintf("You can upload only %d more image(s)", remainingSlots))
		return
	}

	var allImages []ImageMeta

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Println("Error opening file:", err)
			continue
		}

		uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
			PublicID: fileHeader.Filename,
			Folder:   "treeqr/" + treeID,
		})
		file.Close()

		if err != nil {
			log.Println("Cloudinary upload error:", err)
			continue
		}

		allImages = append(allImages, ImageMeta{
			URL:       uploadResult.SecureURL,
			ImageType: imageType,
		})
	}

	if len(allImages) == 0 {
		writeAlert(w, "Failed to upload any images")
		return
	}

	// Save to Firebase
	updatedImages := append(existing, allImages...)
	if err := ref.Set(ctx, updatedImages); err != nil {
		writeAlert(w, "Failed to update image list in Firebase: "+err.Error())
		return
	}

	http.Redirect(w, r, "/image?id="+treeID, http.StatusSeeOther)
}

// Helper to send alert and go back
func writeAlert(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<script>alert(%q); window.history.back();</script>`, msg)
}
