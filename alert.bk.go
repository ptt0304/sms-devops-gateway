package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Receivers   []struct {
		Name string `json:"name"`
	} `json:"receiver"`
}

func HandleAlert(cfg *config.Config, logFile *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// ✅ Ghi toàn bộ alert vào file log
		logEntry := fmt.Sprintf("[%s] Received alert:\n%s\n\n", time.Now().Format(time.RFC3339), string(body))
		if _, err := logFile.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to write to alert log: %v\n", err)
		}

		// Parse alert JSON
		var alert Alert
		if err := json.Unmarshal(body, &alert); err != nil {
			http.Error(w, "invalid JSON object", http.StatusBadRequest)
			return
		}

		alertstate := alert.Labels["alertstate"]
		severity := alert.Labels["severity"]

		if !((alertstate == "resolved") || (alertstate == "firing" && severity == "none")) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Alert ignored"))
			return
		}

		cluster := alert.Labels["cluster"]
		namespace := alert.Labels["namespace"]
		job := alert.Labels["job"]
		summary := alert.Annotations["summary"]

		if alertstate == "" {
			alertstate = "unknown-alertstate"
		}
		if cluster == "" {
			cluster = "unknown-cluster"
		}
		if namespace == "" {
			namespace = "unknown-namespace"
		}
		if job == "" {
			job = "unknown-job"
		}
		if summary == "" {
			summary = alert.Labels["alertname"]
		}

		message := fmt.Sprintf("[%s] %s/%s | %s | %s", alertstate, cluster, namespace, job, summary)

		targetReceiver := ""
		if len(alert.Receivers) > 0 {
			targetReceiver = alert.Receivers[0].Name
		}

		sent := false
		for _, receiver := range cfg.Receivers {
			if receiver.Name == targetReceiver {
				forwarder.SendToMultipleMobiles(receiver.Mobiles, message)
				sent = true
				break
			}
		}
		if !sent {
			forwarder.SendToMultipleMobiles(cfg.DefaultReceiver.Mobiles, message)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Alert processed✅"))
	}
}
