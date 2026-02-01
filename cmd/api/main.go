package main

import (
	"log"
	"net/http"
	"os"

	"github.com/shunto/image-processing-k8s/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", handler.HealthCheck)
	mux.HandleFunc("GET /ready", handler.ReadyCheck)

	// Image processing endpoints
	mux.HandleFunc("POST /api/resize", handler.Resize)
	mux.HandleFunc("POST /api/grayscale", handler.Grayscale)
	mux.HandleFunc("POST /api/rotate", handler.Rotate)

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
