package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Receivers   []struct {
		Name string `json:"name"`
	} `json:"receivers"`
}

// HandleAlert receives a single alert JSON object and sends SMS based on config
func HandleAlert(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var alert Alert
		if err := json.Unmarshal(body, &alert); err != nil {
			http.Error(w, "invalid JSON object", http.StatusBadRequest)
			return
		}

		alertstate := alert.Labels["alertstate"]
		severity := alert.Labels["severity"]
		// Chỉ xử lý nếu alertstate là "firing" và severity là "critical"
		if !((alertstate == "resolved") || (alertstate == "firing" && severity == "none")) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Alert ignored"))
			return
		}

		// Format thông báo gửi SMS
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

		// Gửi tới receiver tương ứng
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
