package treehandler

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

/* ---------------------------- 🔧 Structs ---------------------------- */

type TreeItem struct {
	ID          string
	Family      string
	Botanical   string
	Common      string
	Volunteer   string
	Site        string
	Published   bool
	PublishTime string
	ImageURL    string
	Lat         float64
	Lng         float64
}

type AdminDashboardData struct {
	TreeCount      int
	VolunteerCount int
	SiteCount      int
	CurrentTime    string
	Trees          []TreeItem
}

/* ---------------------- 🚀 Admin Handler ----------------------- */

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// ✅ Firebase init
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
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		log.Println("app.Database:", err)
		return
	}

	// 🌳 Fetch trees
	var trees map[string]interface{}
	if err := db.NewRef("trees").Get(ctx, &trees); err != nil {
		log.Println("db.Get trees:", err)
		trees = make(map[string]interface{})
	}

	// 👥 Fetch volunteers
	volunteerSet := make(map[string]bool)
	var verifiedVolunteers map[string]map[string]interface{}
	if err := db.NewRef("volunteers/verified").Get(ctx, &verifiedVolunteers); err != nil {
		log.Println("Failed to fetch volunteers/verified:", err)
		verifiedVolunteers = map[string]map[string]interface{}{}
	}

	volunteers := make(map[string]bool)
	for _, v := range verifiedVolunteers {
		email := toString(v["email"])
		if email != "" {
			volunteers[email] = true
		}
	}

	// 📊 Collect stats
	treeList := []TreeItem{}
	siteSet := make(map[string]bool)

	for id, raw := range trees {
		treeMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		classification := toMap(treeMap["classification"])
		loc := toMap(treeMap["location"])

		family := toString(classification["family"])
		botanical := toString(treeMap["botanical"])
		common := toString(treeMap["Name"])
		if common == "" {
			common = "Unknown"
		}

		volunteer := toString(treeMap["volunteerName"])
		site := toString(loc["site"])
		published := toBool(treeMap["Published"])
		publishTime := toString(treeMap["timestamp"])
		parsedTime, err := time.Parse(time.RFC3339, publishTime)
		if err == nil {
			publishTime = parsedTime.Format("02 Jan 2006, 03:04 PM")
		}

		var lat, lng float64
		if coordStr, ok := loc["coordinates"].(string); ok && coordStr != "" {
			parts := strings.Split(coordStr, ",")
			if len(parts) == 2 {
				lat, _ = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
				lng, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		}

		var image string

		if imgs, ok := treeMap["images"].([]interface{}); ok && len(imgs) > 0 {
			if imap, ok := imgs[0].(map[string]interface{}); ok {
				image = toString(imap["url"])
			}
		}

		if site != "" {
			siteSet[site] = true
		}

		if volunteer != "" {
			volunteerSet[volunteer] = true
		}

		treeList = append(treeList, TreeItem{
			ID:          id,
			Family:      family,
			Botanical:   botanical,
			Common:      common,
			Volunteer:   volunteer,
			Site:        site,
			Published:   published,
			PublishTime: publishTime,
			ImageURL:    image,
			Lat:         lat,
			Lng:         lng,
		})

	}

	// 📦 Send data to HTML
	data := AdminDashboardData{
		TreeCount:      len(treeList),
		VolunteerCount: len(volunteers),
		SiteCount:      len(siteSet),
		CurrentTime:    time.Now().Format("02 Jan 2006, 03:04 PM"),
		Trees:          treeList,
	}

	tmpl := template.New("admin_dashboard.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) template.JS {
			a, err := json.Marshal(v)
			if err != nil {
				log.Printf("json.Marshal error: %v", err)
				return ""
			}
			return template.JS(a)
		},
	})

	tmpl, err = tmpl.ParseFiles("static/admin_dashboard.html")
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		log.Println("template.ParseFiles:", err)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("tmpl.Execute:", err)
	}

}
