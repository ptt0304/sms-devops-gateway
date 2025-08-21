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
	// Load config chính
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// ✅ Load ignore-alert.json
	ignoreCfg, err := config.LoadIgnoreConfig("ignore-alert.json")
	if err != nil {
		log.Fatalf("❌ Failed to load ignore config: %v", err)
	}

	/////////////////////////////////////////////////////////////////
	// Mở file alerts.log để ghi liên tục
	logFilePath := "/log/alerts.log"
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Cannot open log file: %v", err)
	}
	defer logFile.Close()

	// Ghi log khởi động
	logFile.WriteString("=== SMS DevOps Gateway started ===\n")

	/////////////////////////////////////////////////////////////////
	// ✅ Truyền cả cfg và ignoreCfg vào handler
	http.HandleFunc("/sms", handler.HandleAlert(cfg, ignoreCfg, logFile))

	fmt.Println("🚀 SMS DevOps Gateway running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
