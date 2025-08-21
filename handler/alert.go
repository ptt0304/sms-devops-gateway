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

// AlertData đại diện cho JSON từ Alertmanager
type AlertData struct {
	Receiver string  `json:"receiver"`
	Alerts   []Alert `json:"alerts"`
}

type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// ✅ Hàm kiểm tra alert có bị ignore hay không
func shouldIgnore(cluster, namespace, pod string, ignoreCfg *config.IgnoreConfig) bool {
	for _, c := range ignoreCfg.Ignore {
		if c.Cluster == cluster {
			for _, ns := range c.Namespaces {
				if ns.Name == namespace {
					// Nếu pods rỗng => ignore tất cả pod trong namespace
					if len(ns.Pods) == 0 {
						return true
					}
					// Nếu có danh sách pods => kiểm tra pod cụ thể
					for _, p := range ns.Pods {
						if p == pod {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// ✅ Hàm xử lý alert
func HandleAlert(cfg *config.Config, ignoreCfg *config.IgnoreConfig, logFile *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Ghi toàn bộ alert vào file log
		logEntry := fmt.Sprintf("[%s] Received alert:\n%s\n\n", time.Now().Format(time.RFC3339), string(body))
		if _, err := logFile.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to write to alert log: %v\n", err)
		}

		// Parse alert JSON
		var alertData AlertData
		if err := json.Unmarshal(body, &alertData); err != nil {
			http.Error(w, "invalid JSON object", http.StatusBadRequest)
			return
		}

		if len(alertData.Alerts) == 0 {
			http.Error(w, "no alerts found", http.StatusBadRequest)
			return
		}

		alert := alertData.Alerts[0] // chỉ xử lý alert đầu tiên

		status := alert.Status
		severity := alert.Labels["severity"]
		cluster := alert.Labels["cluster"]
		namespace := alert.Labels["namespace"]
		pod := alert.Labels["pod"]
		summary := alert.Annotations["summary"]

		if status == "" {
			status = "unknown-status"
		}
		if cluster == "" {
			cluster = "unknown-cluster"
		}
		if namespace == "" {
			namespace = "unknown-namespace"
		}
		if pod == "" {
			pod = "unknown-pod"
		}
		if summary == "" {
			summary = alert.Labels["alertname"]
		}

		// ✅ Kiểm tra ignore dựa trên cluster/namespace/pod
		if shouldIgnore(cluster, namespace, pod, ignoreCfg) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Alert ignored by rules for %s/%s/%s ✅", cluster, namespace, pod)))
			return
		}

		// Kiểm tra điều kiện gửi alert như cũ
		if !((status == "resolved") || (status == "firing" && severity == "critical")) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Alert ignored by default rules"))
			return
		}

		message := fmt.Sprintf("[%s] %s/%s | %s | %s", status, cluster, namespace, pod, summary)
		targetReceiver := alertData.Receiver

		// Forward alert đến đúng receiver
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
		w.Write([]byte("Alert processed ✅"))
	}
}
