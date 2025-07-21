package treehandler

import (
	"context"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

type VolunteerInfo struct {
	Email      string `json:"email"`
	ApprovedBy string `json:"approved_by"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Permanent  bool   `json:"permanent"`
}

// Firebase Initialization
func firebaseClient(ctx context.Context) *db.Client {
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	config := &firebase.Config{DatabaseURL: "https://tree-test-f912a-default-rtdb.firebaseio.com"}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		panic("Failed to initialize Firebase App: " + err.Error())
	}
	client, err := app.Database(ctx)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
	return client
}

// POST: /update_volunteer
func UpdateVolunteerPermission(c *gin.Context) {
	ctx := context.Background()
	client := firebaseClient(ctx)

	var update struct {
		Email     string `json:"email"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Permanent bool   `json:"permanent"`
	}
	if err := c.BindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ref := client.NewRef("volunteers")
	var raw map[string]map[string]interface{}
	if err := ref.Get(ctx, &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, data := range raw {
		if data["email"] == update.Email {
			userRef := ref.Child(key)
			updates := map[string]interface{}{
				"start_time": update.StartTime,
				"end_time":   update.EndTime,
				"permanent":  update.Permanent,
			}
			if err := userRef.Update(ctx, updates); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Volunteer updated"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer not found"})
}

// GET: /revoke_volunteer?email=abc@example.com
func RevokeVolunteer(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing email"})
		return
	}

	ctx := context.Background()
	client := firebaseClient(ctx)
	ref := client.NewRef("volunteers")

	var raw map[string]map[string]interface{}
	if err := ref.Get(ctx, &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, data := range raw {
		if data["email"] == email {
			userRef := ref.Child(key)
			if err := userRef.Update(ctx, map[string]interface{}{
				"permission": false,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Permission revoked"})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer not found"})
}
