package treehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"google.golang.org/api/option"
)

// Tree structure
// (kept for potential future reads)
type Tree struct {
	ID        string `json:"ID"`
	Name      string `json:"Name"`
	Timestamp string `json:"Timestamp"`
	QR        bool   `json:"QR"`
	Saved     bool   `json:"Saved"`
	Published bool   `json:"Published"`
	Volunteer string `json:"Volunteer"`
}

// sanitizeName converts a string to a lowercase slug (alphanumerics + "-")
func sanitizeName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := re.ReplaceAllString(name, "-")
	return strings.ToLower(strings.Trim(slug, "-"))
}

// generate4DigitID returns a zero‑padded random 4‑digit string
func generate4DigitID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

/* ────────────────────── Core: create tree + QR ────────────────────── */

func GenerateTreeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "static/qr.html")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Payload definition for JSON requests
	type Payload struct {
		Name string `json:"name"`
		UID  string `json:"uid"`
	}

	var (
		treeName string
		uid      string
		fromForm = true // default to form post
	)

	// Check form field first
	treeName = strings.TrimSpace(r.FormValue("treeName"))

	// Fallback to JSON body
	if treeName == "" {
		fromForm = false
		var payload Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Tree name is required (both form and JSON empty)", http.StatusBadRequest)
			return
		}
		treeName = strings.TrimSpace(payload.Name)
		uid = strings.TrimSpace(payload.UID)
	}

	if treeName == "" {
		http.Error(w, "Tree name is empty", http.StatusBadRequest)
		return
	}

	/* Firebase bootstrapping */
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com",
	}, opt)
	if err != nil {
		http.Error(w, "Firebase initialisation failed", http.StatusInternalServerError)
		return
	}
	client, err := app.Database(context.Background())
	if err != nil {
		http.Error(w, "Firebase DB client creation failed", http.StatusInternalServerError)
		return
	}

	/* JSON‑only mode: overwrite existing UID with a new slug */
	if !fromForm {
		if uid == "" {
			http.Error(w, "UID missing in payload", http.StatusBadRequest)
			return
		}
		ref := client.NewRef("trees/" + uid)
		if err := ref.Set(context.Background(), map[string]string{"ID": sanitizeName(treeName)}); err != nil {
			http.Error(w, "Failed to save tree ID to existing UID", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Tree ID stored successfully"}`))
		return
	}

	/* Form mode: new UID / full entry */
	uid = uuid.New().String()
	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")

	slug := sanitizeName(treeName)
	customID := fmt.Sprintf("%s_%s-%s", treeName, slug, generate4DigitID())

	dbData := map[string]interface{}{
		"ID":            customID,
		"Name":          treeName,
		"Published":     false,
		"QR":            true,
		"Saved":         true,
		"botanical":     "",
		"timestamp":     timestamp,
		"uid":           uid,
		"volunteerName": "sunny21@gmail.com",
	}

	ref := client.NewRef("trees/" + uid)
	if err := ref.Set(context.Background(), dbData); err != nil {
		http.Error(w, "Failed to save full tree data", http.StatusInternalServerError)
		return
	}

	/* Confirmation page */
	viewData := map[string]interface{}{
		"ID":        uid,
		"Name":      treeName,
		"Timestamp": timestamp,
	}
	tmpl, err := template.ParseFiles("static/display.html")
	if err != nil {
		http.Error(w, "Template rendering failed", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, viewData)
}

/* ────────────────────── Static QR endpoint ────────────────────── */

func ServeQRHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing tree ID", http.StatusBadRequest)
		return
	}
	qrURL := fmt.Sprintf("https://geo-tagging-user.onrender.com/%s", id)
	png, err := qrcode.Encode(qrURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "QR generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

/* ────────────────────── PDF download with QR ────────────────────── */

func DownloadPDFHandler(w http.ResponseWriter, r *http.Request) {
	// Accept both ?uid= and legacy ?id=
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		uid = r.URL.Query().Get("id")
	}
	name := r.URL.Query().Get("name")
	if uid == "" || name == "" {
		http.Error(w, "Missing uid or name", http.StatusBadRequest)
		return
	}

	qrURL := fmt.Sprintf("https://geo-tagging-user.onrender.com/%s", uid)
	qrBytes, err := qrcode.Encode(qrURL, qrcode.Medium, 300)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	pageWidth, _ := pdf.GetPageSize()
	pdf.SetY(63)
	pdf.CellFormat(0, 8, name, "", 1, "C", false, 0, "")

	opt := gofpdf.ImageOptions{ImageType: "PNG"}
	pdf.RegisterImageOptionsReader("qr.png", opt, bytes.NewReader(qrBytes))

	qrWidth := 130.0
	x := (pageWidth - qrWidth) / 2
	pdf.ImageOptions("qr.png", x, 70, qrWidth, 0, false, opt, 0, "")

	pdf.SetY(190)
	pdf.CellFormat(0, 10, "Know the Tree, Scan to See!!", "", 1, "C", false, 0, "")

	filename := fmt.Sprintf("%s_%s.pdf", name, uid)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if err := pdf.Output(w); err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
	}
}
