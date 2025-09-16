package handler

import (
	"fmt"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

// Xử lý alert từ Alertmanager
func processK8sAlert(alertData AlertData, cfg *config.Config, ignoreCfg *config.IgnoreConfig, w http.ResponseWriter, logFile *os.File) {
	alert := alertData.Alerts[0]

	status := alert.Status
	cluster := defaultIfEmpty(alert.Labels["cluster"], "unknown-cluster")
	namespace := defaultIfEmpty(alert.Labels["namespace"], "unknown-namespace")
	pod := defaultIfEmpty(alert.Labels["pod"], "unknown-pod")
	severity := alert.Labels["severity"]
	summary := alert.Annotations["summary"]

	// Format đặc biệt cho alert-d1-lgc-devops
	alertname := defaultIfEmpty(alert.Labels["alertname"], "unknown-alertname")
	consumergroup := defaultIfEmpty(alert.Labels["consumergroup"], "unknown-consumergroup")
	job := defaultIfEmpty(alert.Labels["job"], "unknown-job")
	topic := defaultIfEmpty(alert.Labels["topic"], "unknown-topic")

	if summary == "" {
		summary = alert.Labels["alertname"]
	}

	// Rule check
	if !(status == "resolved" || (status == "firing" && severity == "critical")) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Alert ignored by default rules"))
		return
	}

	// Receiver
	targetReceiver := alertData.Receiver
	message := ""

	if targetReceiver == "alert-d1-lgc-devops" {

		message = fmt.Sprintf("[%s] %s | ConsumerGroup: %s | Job: %s | Topic: %s | Sum: %s",
			status, alertname, consumergroup, job, topic, summary)
	} else {

		if topic != "unknown-topic" || consumergroup != "unknown-consumergroup" {
			message = fmt.Sprintf("[%s] %s | ConsumerGroup: %s | Job: %s | Topic: %s | Sum: %s",
			status, alertname, consumergroup, job, topic, summary)
		} else {
			// Format mặc định
			message = fmt.Sprintf("[%s] %s/%s | %s | %s",
				status, cluster, namespace, pod, summary)
		}
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
	w.Write([]byte("Alert processed ✅"))
}
