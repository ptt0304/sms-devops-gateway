package handler

import (
	"fmt"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

// Xử lý alert từ Alertmanager (K8s)
func processK8sAlert(alertData AlertData, cfg *config.Config, ignoreCfg *config.IgnoreConfig, w http.ResponseWriter, logFile *os.File) {
	alert := alertData.Alerts[0]

	status := alert.Status
	cluster := defaultIfEmpty(alert.Labels["cluster"], "unknown-cluster")
	namespace := defaultIfEmpty(alert.Labels["namespace"], "unknown-namespace")
	pod := defaultIfEmpty(alert.Labels["pod"], "unknown-pod")
	severity := alert.Labels["severity"]
	summary := alert.Annotations["summary"]
	if summary == "" {
		summary = alert.Labels["alertname"]
	}

	// Ignore check
	if ignored, reason := shouldIgnore(cluster, namespace, pod, ignoreCfg); ignored {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Alert ignored: %s ✅", reason)))
		logToFile(logFile, fmt.Sprintf("Ignored: %s (%s/%s/%s)", reason, cluster, namespace, pod))
		return
	}

	// Rule check
	if !(status == "resolved" || (status == "firing" && severity == "critical")) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Alert ignored by default rules"))
		return
	}

	// Format message
	message := fmt.Sprintf("[%s] %s/%s | %s | %s", status, cluster, namespace, pod, summary)

	// Receiver
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
