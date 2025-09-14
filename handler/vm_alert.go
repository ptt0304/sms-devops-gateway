package handler

import (
	"fmt"
	"net/http"
	"os"
	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

// Xử lý alert từ VM
func processVMAlert(vmAlert VMAlert, cfg *config.Config, w http.ResponseWriter, logFile *os.File) {
	receiver := "alert-devops"
	summary := vmAlert.Annotations["summary"]
	if summary == "" {
		summary = vmAlert.Name
	}

	severity := vmAlert.Labels["severity"]
	state := vmAlert.State

	// ⚠️ Chỉ gửi nếu firing + critical
	if !(state == "resolved" || (state == "firing" && severity == "critical")) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("VM Alert ignored by default rules"))
		return
	}

	message := fmt.Sprintf("[%s] %s", state, summary)

	// Receiver
	sent := false
	for _, rcv := range cfg.Receivers {
		if rcv.Name == receiver {
			forwarder.SendToMultipleMobiles(rcv.Mobiles, message)
			sent = true
			break
		}
	}
	if !sent {
		forwarder.SendToMultipleMobiles(cfg.DefaultReceiver.Mobiles, message)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("VM Alert processed ✅"))
}
