package treehandler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/fogleman/gg"
	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"google.golang.org/api/option"
)

// ✅ Embed the Roboto font
//
//go:embed Roboto.ttf
var robotoTTF []byte

// ✅ Slugify function to create clean IDs
func slugify(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := re.ReplaceAllString(name, "-")
	return strings.ToLower(strings.Trim(slug, "-"))
}

// ✅ Random 4-digit suffix
func randomSuffix() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

// ✅ Main QR generation handler
func GenerateDirectQR(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]
	if uid == "" {
		http.Error(w, "Missing UID", http.StatusBadRequest)
		return
	}

	// ✅ Firebase init
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com",
	}, opt)
	if err != nil {
		http.Error(w, "Firebase init error", http.StatusInternalServerError)
		return
	}
	client, err := app.Database(context.Background())
	if err != nil {
		http.Error(w, "Firebase client error", http.StatusInternalServerError)
		return
	}

	// ✅ Fetch tree data
	var treeData map[string]interface{}
	ref := client.NewRef("trees/" + uid)
	if err := ref.Get(context.Background(), &treeData); err != nil {
		http.Error(w, "Failed to fetch tree data", http.StatusInternalServerError)
		return
	}

	name, ok := treeData["Name"].(string)
	if !ok || name == "" {
		log.Printf("Tree name missing or not a string for UID: %s", uid)
		http.Error(w, "Tree name missing", http.StatusBadRequest)
		return
	}

	// ✅ Check or generate treeID
	treeID, exists := treeData["ID"].(string)
	if !exists || treeID == "" {
		base := slugify(name)
		suffix := randomSuffix()
		treeID = fmt.Sprintf("%s-%s", base, suffix)
		_ = ref.Update(context.Background(), map[string]interface{}{"ID": treeID})
	}

	qrURL := fmt.Sprintf("https://geo-tagging-user.onrender.com/%s", uid)

	// ✅ Generate QR PNG
	qrPNG, err := qrcode.Encode(qrURL, qrcode.Medium, 256)
	if err != nil {
		log.Printf("QR generation error: %v", err)
		http.Error(w, "QR generation failed", http.StatusInternalServerError)
		return
	}

	// ✅ Prepare image canvas
	const imgWidth, imgHeight = 300, 360
	dc := gg.NewContext(imgWidth, imgHeight)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// ✅ Load font
	fontParsed, err := opentype.Parse(robotoTTF)
	if err != nil {
		log.Println("Failed to parse embedded font:", err)
		http.Error(w, "Font parsing failed", http.StatusInternalServerError)
		return
	}
	face, err := opentype.NewFace(fontParsed, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Println("Failed to create font face:", err)
		http.Error(w, "Font face creation failed", http.StatusInternalServerError)
		return
	}
	dc.SetFontFace(face)

	// ✅ Draw tree name (center wrapped if long)
	dc.SetRGB(0, 0, 0)
	dc.DrawStringWrapped(name, imgWidth/2, 25, 0.5, 0.5, imgWidth-20, 1.5, gg.AlignCenter)

	// ✅ Draw QR image below
	img, _ := png.Decode(bytes.NewReader(qrPNG))
	dc.DrawImageAnchored(img, imgWidth/2, imgHeight/2+20, 0.5, 0.5)

	// ✅ Encode image to base64 PNG
	var finalBuf bytes.Buffer
	if err := png.Encode(&finalBuf, dc.Image()); err != nil {
		http.Error(w, "Image encoding failed", http.StatusInternalServerError)
		return
	}
	qrBase64 := base64.StdEncoding.EncodeToString(finalBuf.Bytes())

	// ✅ Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"treeID":   treeID,
		"qrBase64": qrBase64,
		"qrURL":    qrURL,
		"uid":      uid,
	})
}
