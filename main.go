package main

import (
	"log"

	"financial-planner-app/server"
)

func main() {
	log.Println("Starting Financial Planner App...")

	// Initialize and start server
	srv, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Println("Server initialized, starting on :8080...")
	if err := srv.Run(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
