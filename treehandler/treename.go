package treehandler

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PlantNetResponse struct {
	Results []struct {
		Score   float64 `json:"score"`
		Species struct {
			ScientificNameWithoutAuthor string   `json:"scientificNameWithoutAuthor"`
			CommonNames                 []string `json:"commonNames"`
		} `json:"species"`
	} `json:"results"`
}

type TreeSuggestion struct {
	ScientificName string
	CommonNames    []string
	Score          float64
}

type ParsedTreeInfo struct {
	CommonName    string
	Medicinal     []string
	Environmental []string
}

type SuggestionPageData struct {
	UID         string
	Suggestions []TreeSuggestion
}

func IdentifyHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	volunteerName := r.FormValue("volunteerName")
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image upload failed", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var imgBuf bytes.Buffer
	if _, err := io.Copy(&imgBuf, file); err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// STEP 1: Upload to Cloudinary with signature
	cloudName := "dybnayjf6"
	apiKey := "676827133671535"
	apiSecret := "ZM8oGKcKNsXzaTjHZ27GRGRQ7CE"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	signParams := fmt.Sprintf("timestamp=%s%s", timestamp, apiSecret)
	h := sha1.New()
	h.Write([]byte(signParams))
	signature := hex.EncodeToString(h.Sum(nil))

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("file", "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(imgBuf.Bytes()))
	writer.WriteField("timestamp", timestamp)
	writer.WriteField("api_key", apiKey)
	writer.WriteField("signature", signature)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to upload to Cloudinary", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	cloudBody, _ := io.ReadAll(res.Body)
	log.Println("Cloudinary response:", string(cloudBody))

	var cloudResp struct {
		SecureURL string `json:"secure_url"`
	}
	if err := json.Unmarshal(cloudBody, &cloudResp); err != nil || cloudResp.SecureURL == "" {
		http.Error(w, "Invalid Cloudinary response", http.StatusInternalServerError)
		return
	}

	// STEP 2: Call PlantNet API
	var plantBuf bytes.Buffer
	plantWriter := multipart.NewWriter(&plantBuf)
	plantPart, _ := plantWriter.CreateFormFile("images", "upload.jpg")
	plantPart.Write(imgBuf.Bytes())
	plantWriter.WriteField("organs", "leaf")
	plantWriter.Close()

	plantAPIKey := strings.TrimSpace("2b10KkzCQITYJDyaUiEMGCDVtu")
	plantReq, _ := http.NewRequest("POST", "https://my-api.plantnet.org/v2/identify/all?api-key="+plantAPIKey, &plantBuf)
	plantReq.Header.Set("Content-Type", plantWriter.FormDataContentType())

	plantResp, err := http.DefaultClient.Do(plantReq)
	if err != nil {
		http.Error(w, "Error contacting PlantNet", http.StatusInternalServerError)
		return
	}
	defer plantResp.Body.Close()

	bodyBytes, _ := io.ReadAll(plantResp.Body)
	plantResp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var plantNetRes PlantNetResponse
	if err := json.NewDecoder(plantResp.Body).Decode(&plantNetRes); err != nil {
		http.Error(w, "Failed to decode PlantNet response", http.StatusInternalServerError)
		log.Println("Decode error:", err)
		return
	}

	if len(plantNetRes.Results) == 0 {
		http.Error(w, "No results found. Try another image.", http.StatusNotFound)
		return
	}

	// STEP 3: Save metadata to Firebase (excluding names)
	uid := uuid.New().String()

	data := map[string]interface{}{
		"uid":           uid,
		"volunteerName": volunteerName,
		"Saved":         false,
		"Published":     false,
		"QR":            false,
		"timestamp":     time.Now().Format(time.RFC3339),
		"images": []map[string]string{
			{
				"imageType": "tree",
				"url":       cloudResp.SecureURL,
			},
		},
	}

	jsonData, _ := json.Marshal(data)
	firebaseURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s.json", uid)
	reqFB, _ := http.NewRequest("PUT", firebaseURL, bytes.NewBuffer(jsonData))
	reqFB.Header.Set("Content-Type", "application/json")

	respFB, err := http.DefaultClient.Do(reqFB)
	if err != nil {
		http.Error(w, "Error saving to Firebase", http.StatusInternalServerError)
		log.Println("Firebase error:", err)
		return
	}
	defer respFB.Body.Close()

	// STEP 4: Display suggestions
	var suggestions []TreeSuggestion
	for i, res := range plantNetRes.Results {
		if i >= 10 {
			break
		}
		suggestions = append(suggestions, TreeSuggestion{
			ScientificName: res.Species.ScientificNameWithoutAuthor,
			CommonNames:    res.Species.CommonNames,
			Score:          res.Score * 100,
		})
	}
	tmpl, err := template.ParseFiles("static/suggestion.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, SuggestionPageData{
		UID:         uid,
		Suggestions: suggestions,
	})

}

func GetTreeDetailsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	treeName := r.FormValue("scientificName")
	commonName := r.FormValue("commonNames")
	uid := r.FormValue("uid")

	if treeName == "" || uid == "" || commonName == "" {
		http.Error(w, "Missing data for Firebase write", http.StatusBadRequest)
		return
	}

	// Update Firebase with common name and scientific name
	updateData := map[string]interface{}{
		"uid":       uid,
		"Name":      commonName,
		"botanical": treeName,
		"Saved":     true,
		"QR":        true, // ✅ added this line
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		http.Error(w, "Failed to marshal Firebase data", http.StatusInternalServerError)
		return
	}

	firebaseURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s.json", uid)
	req, err := http.NewRequest("PATCH", firebaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to create Firebase request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to update Firebase", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Fetch AI description
	fullText := fetchTreeInfoFromOpenRouter(treeName)

	// Fetch images from Firebase
	imageFetchURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s/images.json", uid)
	imageResp, err := http.Get(imageFetchURL)
	if err != nil {
		http.Error(w, "Failed to fetch images from Firebase", http.StatusInternalServerError)
		return
	}
	defer imageResp.Body.Close()

	var images []struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(imageResp.Body).Decode(&images); err != nil {
		http.Error(w, "Failed to parse image data", http.StatusInternalServerError)
		return
	}

	// Extract just the URLs
	var imageURLs []string
	for _, img := range images {
		imageURLs = append(imageURLs, img.URL)
	}

	// Final struct to send to HTML
	data := struct {
		ScientificName string
		ResponseText   string
		UID            string
		ImageURLs      []string
	}{
		ScientificName: treeName,
		ResponseText:   fullText,
		UID:            uid,
		ImageURLs:      imageURLs,
	}

	tmpl, err := template.ParseFiles("static/treeinfo.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func fetchTreeInfoFromOpenRouter(name string) string {
	apiKey := "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"

	prompt := fmt.Sprintf(`You are a knowledgeable botanical expert. Based only on accurate botanical taxonomy, what is the most widely accepted scientific and common name for the plant with the scientific name '%s'? Do not confuse it with similar plants.

Include:
- Common Name:
- Detailed Description: (Provide a comprehensive and detailed paragraph explaining the plant’s characteristics, appearance, growth behavior, and typical habitat)
- Medicinal Benefits: (Give an in-depth explanation of traditional and modern medicinal uses, including any active compounds if known)
- Environmental Benefits: (Provide detailed benefits this plant offers to the ecosystem — such as carbon absorption, air purification, soil enrichment, biodiversity support, etc.)
- Native to India: Yes or No
- Scientific Classification:
  - Kingdom:
  - Phylum (or Division for plants):
  - Class:
  - Order:
  - Family:
  - Genus:
  - Species:
- Common Tree Category: (Return only one from this list exactly as is, without any extra explanation)
  - Medicinal Trees
  - Fruit-Bearing Trees
  - Timber Trees
  - Ornamental Trees
  - Shade Trees
  - Sacred or Religious Trees
  - Evergreen Trees
  - Deciduous Trees
  - Endangered
  - Rare Trees
  - Others`, name)

	reqBody := map[string]interface{}{
		"model": "mistralai/mistral-7b-instruct:free",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful botanical expert."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 700,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("OpenRouter request failed:", err)
		return "Error contacting OpenRouter."
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil || len(result.Choices) == 0 {
		log.Println("Failed to parse OpenRouter response")
		log.Println(string(body))
		return "Failed to parse OpenRouter response."
	}

	return result.Choices[0].Message.Content
}

func AppendImageHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	uid := r.FormValue("uid")
	if uid == "" {
		http.Error(w, "UID is required", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image upload failed", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var imgBuf bytes.Buffer
	if _, err := io.Copy(&imgBuf, file); err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// Cloudinary credentials
	cloudName := "dybnayjf6"
	apiKey := "676827133671535"
	apiSecret := "ZM8oGKcKNsXzaTjHZ27GRGRQ7CE"

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signParams := fmt.Sprintf("timestamp=%s%s", timestamp, apiSecret)

	h := sha1.New()
	h.Write([]byte(signParams))
	signature := hex.EncodeToString(h.Sum(nil))

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("file", "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(imgBuf.Bytes()))
	writer.WriteField("timestamp", timestamp)
	writer.WriteField("api_key", apiKey)
	writer.WriteField("signature", signature)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to upload to Cloudinary", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		log.Println("❌ Cloudinary upload error:", string(bodyBytes))
		http.Error(w, "Cloudinary upload failed", http.StatusInternalServerError)
		return
	}

	var cloudResp struct {
		SecureURL string `json:"secure_url"`
	}
	cloudBody, _ := io.ReadAll(res.Body)
	log.Println("Cloudinary response:", string(cloudBody))

	if err := json.Unmarshal(cloudBody, &cloudResp); err != nil || cloudResp.SecureURL == "" {
		http.Error(w, "Invalid Cloudinary response", http.StatusInternalServerError)
		return
	}

	log.Println("✅ Cloudinary URL:", cloudResp.SecureURL)

	// Fetch existing images (array format)
	fetchURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s/images.json", uid)
	respGet, err := http.Get(fetchURL)
	if err != nil {
		http.Error(w, "Failed to fetch existing images", http.StatusInternalServerError)
		return
	}
	defer respGet.Body.Close()

	var existing []map[string]string
	json.NewDecoder(respGet.Body).Decode(&existing)

	if len(existing) >= 4 {
		http.Error(w, "Only 4 images allowed", http.StatusBadRequest)
		return
	}

	// Add the new image
	newImage := map[string]string{
		"imageType": "tree",
		"url":       cloudResp.SecureURL,
	}
	existing = append(existing, newImage)

	updatedJSON, _ := json.Marshal(existing)

	putURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s/images.json", uid)
	putReq, _ := http.NewRequest("PUT", putURL, bytes.NewBuffer(updatedJSON))
	putReq.Header.Set("Content-Type", "application/json")

	respPut, err := http.DefaultClient.Do(putReq)
	if err != nil || respPut.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(respPut.Body)
		log.Println("❌ Firebase PUT error:", string(bodyBytes))
		http.Error(w, "Failed to save image to Firebase", http.StatusInternalServerError)
		return
	}
	defer respPut.Body.Close()

	log.Println("✅ Image added to Firebase")

	// Return JSON with URL
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": cloudResp.SecureURL,
	})
}

func DeleteTreeHandleridentify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid method"})
		return
	}

	var reqData struct {
		UID string `json:"uid"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid JSON"})
		return
	}

	if reqData.UID == "" || reqData.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Missing UID or URL"})
		return
	}

	// Step 1: Fetch image array
	fetchURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s/images.json", reqData.UID)
	resp, err := http.Get(fetchURL)
	if err != nil {
		log.Println("🔥 Failed to fetch Firebase data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to fetch Firebase data"})
		return
	}
	defer resp.Body.Close()

	var images []map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		log.Println("🔥 Failed to decode images array:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to parse images data"})
		return
	}

	// Step 2: Filter out deleted image
	var updatedImages []map[string]string
	var publicID string
	found := false

	for _, img := range images {
		if img["url"] == reqData.URL {
			found = true
			parts := strings.Split(reqData.URL, "/")
			last := parts[len(parts)-1]
			publicID = strings.SplitN(last, ".", 2)[0]
			continue
		}
		updatedImages = append(updatedImages, img)
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Image not found"})
		return
	}

	// Step 3: Delete from Cloudinary
	if publicID != "" {
		apiKey := "676827133671535"
		apiSecret := "ZM8oGKcKNsXzaTjHZ27GRGRQ7CE"
		cloudName := "dybnayjf6"

		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		signString := fmt.Sprintf("public_id=%s&timestamp=%s%s", publicID, timestamp, apiSecret)

		h := sha1.New()
		h.Write([]byte(signString))
		signature := hex.EncodeToString(h.Sum(nil))

		form := url.Values{}
		form.Set("public_id", publicID)
		form.Set("api_key", apiKey)
		form.Set("timestamp", timestamp)
		form.Set("signature", signature)

		deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", cloudName)
		resp, err := http.PostForm(deleteURL, form)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			log.Println("📤 Cloudinary deletion:", string(body))
		}
	}

	// Step 4: Rewrite image list in Firebase
	putURL := fmt.Sprintf("https://treeqrsystem-default-rtdb.firebaseio.com/trees/%s/images.json", reqData.UID)
	data, _ := json.Marshal(updatedImages)
	putReq, _ := http.NewRequest("PUT", putURL, bytes.NewReader(data))
	putReq.Header.Set("Content-Type", "application/json")

	respPut, err := http.DefaultClient.Do(putReq)
	if err != nil {
		log.Println("🔥 Firebase update failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to update Firebase"})
		return
	}
	defer respPut.Body.Close()

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
