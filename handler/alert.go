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

// ✅ Hàm kiểm tra alert có bị ignore hay không và trả về lý do
func shouldIgnore(cluster, namespace, pod string, ignoreCfg *config.IgnoreConfig) (bool, string) {
	// ✅ Ưu tiên kiểm tra wildcard cluster trước
	for _, c := range ignoreCfg.Ignore {
		if c.Cluster == "*" {
			return true, "ignore all alerts due to wildcard cluster"
		}
	}

	// ✅ Sau đó kiểm tra các rule cụ thể
	for _, c := range ignoreCfg.Ignore {
		if c.Cluster == cluster {
			for _, ns := range c.Namespaces {
				if ns.Name == "*" {
					return true, fmt.Sprintf("ignore all alerts in cluster %s due to wildcard namespace", cluster)
				}
				if ns.Name == namespace {
					for _, p := range ns.Pods {
						if p == "*" {
							return true, fmt.Sprintf("ignore all alerts in namespace %s (cluster %s) due to wildcard pod", namespace, cluster)
						}
						if p == pod {
							return true, fmt.Sprintf("ignore alert for specific pod %s in namespace %s (cluster %s)", pod, namespace, cluster)
						}
					}
				}
			}
		}
	}
	return false, ""
}

// ✅ Hàm xử lý alert
func HandleAlert(cfg *config.Config, ignoreCfg *config.IgnoreConfig, logFile *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Đọc body từ request
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

		// Parse JSON alert
		var alertData AlertData
		if err := json.Unmarshal(body, &alertData); err != nil {
			http.Error(w, "invalid JSON object", http.StatusBadRequest)
			return
		}

		if len(alertData.Alerts) == 0 {
			http.Error(w, "no alerts found", http.StatusBadRequest)
			return
		}

		// ✅ Chỉ xử lý alert đầu tiên
		alert := alertData.Alerts[0]

		// ✅ Lấy thông tin từ alert, gán giá trị mặc định nếu thiếu
		status := defaultIfEmpty(alert.Status, "unknown-status")
		cluster := defaultIfEmpty(alert.Labels["cluster"], "unknown-cluster")
		namespace := defaultIfEmpty(alert.Labels["namespace"], "unknown-namespace")
		pod := defaultIfEmpty(alert.Labels["pod"], "unknown-pod")
		severity := alert.Labels["severity"]
		summary := alert.Annotations["summary"]
		if summary == "" {
			summary = alert.Labels["alertname"]
		}

		// ✅ Kiểm tra ignore dựa trên cluster/namespace/pod với wildcard
		if ignored, reason := shouldIgnore(cluster, namespace, pod, ignoreCfg); ignored {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Alert ignored: %s ✅", reason)))
			fmt.Fprintf(os.Stdout, "ℹ️ %s (%s/%s/%s)\n", reason, cluster, namespace, pod)
			return
		}

		// ✅ Kiểm tra điều kiện gửi alert như cũ (chỉ gửi khi resolved hoặc firing+critical)
		if !(status == "resolved" || (status == "firing" && severity == "critical")) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Alert ignored by default rules"))
			return
		}

		// ✅ Tạo nội dung message
		message := fmt.Sprintf("[%s] %s/%s | %s | %s", status, cluster, namespace, pod, summary)
		targetReceiver := alertData.Receiver

		// ✅ Forward alert đến đúng receiver
		sent := false
		for _, receiver := range cfg.Receivers {
			if receiver.Name == targetReceiver {
				forwarder.SendToMultipleMobiles(receiver.Mobiles, message)
				sent = true
				break
			}
		}

		// ✅ Nếu không tìm thấy receiver → gửi đến default
		if !sent {
			forwarder.SendToMultipleMobiles(cfg.DefaultReceiver.Mobiles, message)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Alert processed ✅"))
	}
}

// ✅ Hàm helper để gán giá trị mặc định nếu rỗng
func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
