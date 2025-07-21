package treehandler

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

func AddMissingPublishTimes() {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}, option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json"))
	if err != nil {
		log.Fatalln("Firebase init failed:", err)
	}

	db, err := app.Database(ctx)
	if err != nil {
		log.Fatalln("Firebase DB error:", err)
	}

	var trees map[string]interface{}
	if err := db.NewRef("trees").Get(ctx, &trees); err != nil {
		log.Fatalln("Fetching trees failed:", err)
	}

	for id, raw := range trees {
		tree, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip if already has timestamp
		if _, exists := tree["timestamp"]; exists {
			continue
		}

		// Update with current time
		now := time.Now().Format("02 Jan 2006, 03:04 PM")

		err := db.NewRef("trees/"+id).Update(ctx, map[string]interface{}{
			"timestamp": now,
		})
		if err != nil {
			log.Println("Update failed for", id, ":", err)
		} else {
			fmt.Println("✅ timestamp set for:", id)
		}
	}
}
