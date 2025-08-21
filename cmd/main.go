package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/handler"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
/////////////////////////////////////////////////////////////////
	// Mở file alerts.log để ghi liên tục
	logFilePath := "/log/alerts.log"
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Cannot open log file: %v", err)
	}
	defer logFile.Close()

	// Ghi log khởi động
	logFile.WriteString("=== SMS DevOps Gateway started ===\n")

/////////////////////////////////////////////////////////////////
	// Truyền logFile vào handler
	http.HandleFunc("/sms", handler.HandleAlert(cfg, logFile))

	fmt.Println("🚀 SMS DevOps Gateway running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
