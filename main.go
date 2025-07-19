package main

import (
	"geotagging/routes"
	"log"
	"net/http"
)

func main() {
	r := routes.SetupRoutes()

	log.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
