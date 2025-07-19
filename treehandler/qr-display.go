package treehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	firebase "firebase.google.com/go/v4"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"google.golang.org/api/option"
)

type TreeData struct {
	UID           string `json:"uid"`
	ID            string `json:"ID"` // capital "ID" matches Firebase
	Name          string `json:"Name"`
	Timestamp     string `json:"timestamp"`
	QR            bool   `json:"QR"`
	Saved         bool   `json:"Saved"`
	Published     bool   `json:"Published"`
	VolunteerName string `json:"volunteerName"`
}

func ServeQRHandlertree(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email parameter", http.StatusBadRequest)
		return
	}

	resp, err := http.Get("https://treeqrsystem-default-rtdb.firebaseio.com/trees.json")
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read data", http.StatusInternalServerError)
		return
	}

	var result map[string]TreeData
	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, "Failed to parse data", http.StatusInternalServerError)
		return
	}

	var filteredTrees []TreeData
	for _, tree := range result {
		if tree.Published && tree.QR && strings.EqualFold(tree.VolunteerName, email) {
			filteredTrees = append(filteredTrees, tree)
		}
	}

	tmpl, err := template.ParseFiles("static/qr-display.html")
	if err != nil {
		log.Println("❌ Template parsing failed:", err)
		http.Error(w, "Template parsing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, filteredTrees)
	if err != nil {
		log.Println("❌ Template execution failed:", err)
		http.Error(w, "Template rendering failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

}

func DownloadPDFHandlertree(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("name")

	if id == "" || name == "" {
		http.Error(w, "Missing ID or Name", http.StatusBadRequest)
		return
	}

	// ✅ Use uid in the path
	url := fmt.Sprintf("https://tree-qr.onrender.com/%s", id)

	// Generate QR Code
	qrPNG, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Title
	pdf.CellFormat(0, 15, name, "", 1, "C", false, 0, "")

	// Add QR image
	imgOpts := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   false,
	}
	pdf.RegisterImageOptionsReader("qr.png", imgOpts, bytes.NewReader(qrPNG))
	pdf.ImageOptions("qr.png", 80, 50, 50, 50, false, imgOpts, 0, "")

	// Footer text
	pdf.SetY(120)
	pdf.SetFont("Arial", "", 14)
	pdf.CellFormat(0, 15, "Know the tree. Scan to see!!", "", 1, "C", false, 0, "")

	// Output PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=tree_qr_"+id+".pdf")
	err = pdf.Output(w)
	if err != nil {
		http.Error(w, "Could not write PDF", http.StatusInternalServerError)
	}
}

func DeleteTreeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Missing tree ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}
	app, err := firebase.NewApp(ctx, conf, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Fatalln("Error initializing app:", err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Println("Database client error:", err)
		http.Error(w, "Failed to connect to Realtime DB", http.StatusInternalServerError)
		return
	}

	// 🔍 First: Fetch all trees to find the Firebase key for the given .ID
	var allTrees map[string]TreeData
	if err := client.NewRef("trees").Get(ctx, &allTrees); err != nil {
		log.Println("Error fetching trees:", err)
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	var firebaseKeyToDelete string
	for key, tree := range allTrees {
		if tree.ID == id {
			firebaseKeyToDelete = key
			break
		}
	}

	if firebaseKeyToDelete == "" {
		http.Error(w, "Tree ID not found", http.StatusNotFound)
		return
	}

	// ✅ Delete using Firebase key
	ref := client.NewRef("trees/" + firebaseKeyToDelete)
	if err := ref.Delete(ctx); err != nil {
		log.Println("Delete error:", err)
		http.Error(w, "Failed to delete record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted tree with ID:", id)
	http.Redirect(w, r, "/qr-display?email="+url.QueryEscape(r.FormValue("email")), http.StatusSeeOther)

}
