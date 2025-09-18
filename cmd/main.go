package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/handler"
)

func main() {
	// Load config chính
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	/////////////////////////////////////////////////////////////////
	// Mở file alerts.log để ghi liên tục
	logFilePath := "/log/alerts.log"
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Cannot open log file: %v", err)
	}
	defer logFile.Close()

	// Tạo writer vừa ghi file vừa ghi console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	// Ghi log khởi động
	log.Println("=== SMS DevOps Gateway started ===")
	log.Println("🚀 SMS DevOps Gateway running on :8080")

	/////////////////////////////////////////////////////////////////
	// ✅ Truyền cfg vào handler
	http.HandleFunc("/sms", handler.HandleAlert(cfg, logFile))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
