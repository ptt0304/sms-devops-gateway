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

// nowLocal trả về thời gian hiện tại theo timezone của container (image OS)
func nowLocal() time.Time {
	return time.Now().Local() // dựa vào /etc/localtime trong container
}

// isWithinTimeRange kiểm tra thời gian hiện tại có nằm trong khoảng TimeRange không
func isWithinTimeRange(tr config.TimeRange) bool {
	now := nowLocal()

	if tr.Start != nil && now.Before(*tr.Start) {
		return false
	}
	if tr.End != nil && now.After(*tr.End) {
		return false
	}
	return true
}

// shouldIgnore kiểm tra alert có bị ignore hay không và trả về lý do
func shouldIgnore(cluster, namespace, pod string, ignoreCfg *config.IgnoreConfig) (bool, string) {
	for _, c := range ignoreCfg.Ignore {
		// Check cluster name
		if c.Cluster != "*" && c.Cluster != cluster {
			continue
		}

		// Nếu cluster còn trong time và namespaces rỗng → ignore toàn cluster
		if isWithinTimeRange(c.Time) && len(c.Namespaces) == 0 {
			return true, fmt.Sprintf("ignore all alerts in cluster %s (active time range)", cluster)
		}

		// Check namespace
		for _, ns := range c.Namespaces {
			if ns.Name != "*" && ns.Name != namespace {
				continue
			}

			// Nếu namespace còn trong time và pods rỗng → ignore toàn namespace
			if isWithinTimeRange(ns.Time) && len(ns.Pods) == 0 {
				return true, fmt.Sprintf("ignore all alerts in namespace %s (cluster %s)", namespace, cluster)
			}

			// Check pod
			for _, p := range ns.Pods {
				if p.Name == "*" || p.Name == pod {
					if isWithinTimeRange(p.Time) {
						return true, fmt.Sprintf("ignore alert for pod %s/%s/%s (active time range)", cluster, namespace, pod)
					}
				}
			}
		}
	}
	return false, ""
}

// HandleAlert xử lý alert từ Alertmanager
func HandleAlert(cfg *config.Config, ignoreCfg *config.IgnoreConfig, logFile *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Ghi log theo thời gian local của container
		logEntry := fmt.Sprintf("[%s] Received alert:\n%s\n\n", nowLocal().Format(time.RFC3339), string(body))
		if _, err := logFile.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to write to alert log: %v\n", err)
		}

		var alertData AlertData
		if err := json.Unmarshal(body, &alertData); err != nil {
			http.Error(w, "invalid JSON object", http.StatusBadRequest)
			return
		}

		if len(alertData.Alerts) == 0 {
			http.Error(w, "no alerts found", http.StatusBadRequest)
			return
		}

		alert := alertData.Alerts[0]
		status := defaultIfEmpty(alert.Status, "unknown-status")
		cluster := defaultIfEmpty(alert.Labels["cluster"], "unknown-cluster")
		namespace := defaultIfEmpty(alert.Labels["namespace"], "unknown-namespace")
		pod := defaultIfEmpty(alert.Labels["pod"], "unknown-pod")
		severity := alert.Labels["severity"]
		summary := alert.Annotations["summary"]
		if summary == "" {
			summary = alert.Labels["alertname"]
		}

		if ignored, reason := shouldIgnore(cluster, namespace, pod, ignoreCfg); ignored {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Alert ignored: %s ✅", reason)))
			fmt.Fprintf(os.Stdout, "ℹ️ %s (%s/%s/%s)\n", reason, cluster, namespace, pod)
			return
		}

		if !(status == "resolved" || (status == "firing" && severity == "critical")) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Alert ignored by default rules"))
			return
		}

		message := fmt.Sprintf("[%s] %s/%s | %s | %s", status, cluster, namespace, pod, summary)
		targetReceiver := alertData.Receiver

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

func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
