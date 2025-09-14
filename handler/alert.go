package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"sms-devops-gateway/config"
)

// Dispatcher
func HandleAlert(cfg *config.Config, ignoreCfg *config.IgnoreConfig, logFile *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		logEntry := fmt.Sprintf("[%s] Received alert:\n%s\n\n", time.Now().Format(time.RFC3339), string(body))
		logFile.WriteString(logEntry)

		// Try K8s format
		var alertData AlertData
		if err := json.Unmarshal(body, &alertData); err == nil && len(alertData.Alerts) > 0 {
			if alertData.Alerts[0].Status == "" || alertData.Alerts[0].Labels["severity"] == "" {
				// thiếu status/severity, sẽ rơi xuống http.Error ở dưới
			} else {
				processK8sAlert(alertData, cfg, ignoreCfg, w, logFile)
				return
			}
		}

		// Try VM format
		var vmAlert VMAlert
		if err := json.Unmarshal(body, &vmAlert); err == nil && vmAlert.State != "" {
			if vmAlert.State == "" || vmAlert.Labels["severity"] == "" {
				// thiếu state/severity, sẽ rơi xuống http.Error ở dưới
			} else {
				processVMAlert(vmAlert, cfg, w, logFile)
				return
			}
		}

		http.Error(w, "invalid alert format", http.StatusBadRequest)
	}
}
