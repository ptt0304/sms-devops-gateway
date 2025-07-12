package main

import (
	"fmt"
	"log"
	"net/http"

	"sms-devops-gateway/config"
	"sms-devops-gateway/handler"
)

func main() {
	// Load config.yaml
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Gán handler có nhận config
	http.HandleFunc("/sms", handler.HandleAlert(cfg))

	fmt.Println("🚀 SMS DevOps Gateway running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
