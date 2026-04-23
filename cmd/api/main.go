package main

import (
	"delay-argument-go/cmd/db"
	"delay-argument-go/internal/api"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	// Создаем директорию для результатов
	resultsDir := os.Getenv("RESULTS_DIR")
	if resultsDir == "" {
		resultsDir = "./results"
	}
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatal("Failed to create results directory:", err)
	}

	config := db.LoadConfig()
	s := api.NewServer(config)
	// Настраиваем маршруты
	router := s.SetupRoutes(resultsDir)

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
