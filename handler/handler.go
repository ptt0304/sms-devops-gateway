package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

// Xử lý alert từ Alertmanager
func processAlert(alertData AlertData, cfg *config.Config, w http.ResponseWriter, logFile *os.File) {
	
	// Field chung
	alert := alertData.Alerts[0]
	alertgroup := defaultIfEmpty(alert.Labels["alertgroup"], "unknown-alertgroup")
	alertname := defaultIfEmpty(alert.Labels["alertname"], "unknown-alertname")
	status := alert.Status

	// Field cho k8s
	cluster := defaultIfEmpty(alert.Labels["cluster"], "unknown-cluster")
	namespace := defaultIfEmpty(alert.Labels["namespace"], "unknown-namespace")
	pod := defaultIfEmpty(alert.Labels["pod"], "unknown-pod")
	severity := alert.Labels["severity"]
	summary := alert.Annotations["summary"]

	// Field cho alert-d1-lgc-devops
	consumergroup := defaultIfEmpty(alert.Labels["consumergroup"], "unknown-consumergroup")
	job := defaultIfEmpty(alert.Labels["job"], "unknown-job")
	topic := defaultIfEmpty(alert.Labels["topic"], "unknown-topic")
	instance := defaultIfEmpty(alert.Labels["instance"], "unknown-instance")

	if summary == "" {
		summary = alert.Labels["alertname"]
	}

	/////////////////////////////////////////////////////////////////
	// 📝 Log JSON alert gốc
	alertJSON, _ := json.MarshalIndent(alertData, "", "  ")
	fmt.Fprintf(logFile, "\n📥 Full Alert Received:\n%s\n", string(alertJSON))
	fmt.Printf("\n📥 Full Alert Received:\n%s\n", string(alertJSON))

	/////////////////////////////////////////////////////////////////
	// Rule check
	if !(status == "resolved" || (status == "firing" && severity == "critical")) {
		msg := "⚠️ Alert ignored by default rules"
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(msg))

		fmt.Fprintln(logFile, msg)
		fmt.Println(msg)
		return
	}

	/////////////////////////////////////////////////////////////////
	// Build message
	targetReceiver := alertData.Receiver
	message := ""

	// alert-d1-lgc-devops alert
	if targetReceiver == "alert-d1-lgc-devops" {
			// Instance alert
		if instance != "unknown-instance" {
			message = fmt.Sprintf("[%s] AlertName: %s | Instance: %s | Sum: %s",
				status, alertname, instance, summary)

			// Message queue alert
		} else if topic != "unknown-topic" || consumergroup != "unknown-consumergroup" {
			message = fmt.Sprintf("[%s] %s | ConsumerGroup: %s | Job: %s | Topic: %s | Sum: %s",
				status, alertname, consumergroup, job, topic, summary)
		} else {
			// Missing fields alert
			message = fmt.Sprintf("[%s] Legacy alert type but mising fields | AlertGroup: %s | AlertName: %s | Sum: %s",
				status, alertgroup, alertname, summary)
		}
	
	} else if targetReceiver == "alert-devops" {
		// alert-devops receiver
		if cluster != "unknown-cluster" || namespace != "unknown-namespace" || pod != "unknown-pod"{
			// Missing fields alert
			message = fmt.Sprintf("[%s] %s/%s | %s | %s",
				status, cluster, namespace, pod, summary)
		} else {
			// Missing fields alert
			message = fmt.Sprintf("[%s] K8S alert tyep but missing fields | AlertGroup: %s | AlertName: %s | Sum: %s",
				status, alertgroup, alertname, summary)
		}
	} else {
		//Missing receiver, sent to default_receiver
		message = fmt.Sprintf("[%s] AlertGroup: %s | AlertName: %s | Sum: %s",
			status, alertgroup, alertname, summary)
	}


	// 📝 Log message đã build
	fmt.Fprintf(logFile, "📤 Built message: %s\n", message)
	fmt.Printf("📤 Built message: %s\n", message)

	/////////////////////////////////////////////////////////////////
	// Forward tới receiver
	sent := false
	for _, receiver := range cfg.Receivers {
		if receiver.Name == targetReceiver {
			forwarder.SendToMultipleMobiles(receiver.Mobiles, message)

			fmt.Fprintf(logFile, "📲 Message sent to receiver: %s\n", receiver.Name)
			fmt.Printf("📲 Message sent to receiver: %s\n", receiver.Name)

			sent = true
			break
		}
	}
	if !sent {
		forwarder.SendToMultipleMobiles(cfg.DefaultReceiver.Mobiles, message)

		fmt.Fprintf(logFile, "📲 Message sent to default receiver\n")
		fmt.Printf("📲 Message sent to default receiver\n")
	}

	/////////////////////////////////////////////////////////////////
	// Response cho Alertmanager
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert processed ✅"))
}
