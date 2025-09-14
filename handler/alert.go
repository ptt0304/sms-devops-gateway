package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"sms-devops-gateway/config"
	"sms-devops-gateway/forwarder"
)

type AlertData struct {
	Receiver string  `json:"receiver"`
	Alerts   []Alert `json:"alerts"`
}

type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// Format mới (VM)
type VMAlert struct {
	State       string            `json:"state"`
	Name        string            `json:"name"`
	Value       string            `json:"value"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	ID          string            `json:"id"`
	RuleID      string            `json:"rule_id"`
}

// nowLocal trả về thời gian hiện tại theo timezone của container
func nowLocal() time.Time {
	return time.Now().Local()
}

// isWithinTimeRange kiểm tra nếu now nằm trong time range
func isWithinTimeRange(tr config.TimeRange) bool {
	now := nowLocal()
	if tr.Start != nil && now.Before(*tr.Start) {
		return false
	}
	if tr.End != nil && now.After(*tr.End) {
		return false
	}
	return tr.Start != nil || tr.End != nil
}

// matchWithWildcard hỗ trợ match string với wildcard (*)
func matchWithWildcard(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(pattern, value)
}

// shouldIgnore kiểm tra alert có bị ignore dựa vào các cấp độ và wildcard
func shouldIgnore(cluster, namespace, pod string, ignoreCfg *config.IgnoreConfig) (bool, string) {
	// 1. Check ClusterGroups
	for _, cg := range ignoreCfg.ClusterGroups {
		for _, c := range cg.Clusters {
			if matchWithWildcard(c, cluster) && isWithinTimeRange(cg.Time) {
				return true, fmt.Sprintf("ignored by clusterGroup %s (cluster %s)", cg.Name, cluster)
			}
		}
	}

	// 2. Check từng Cluster
	for _, c := range ignoreCfg.Ignore {
		if !matchWithWildcard(c.Cluster, cluster) {
			continue
		}

		// Nếu cluster còn trong time và không có namespace rule
		if isWithinTimeRange(c.Time) && len(c.Namespaces) == 0 && len(c.NamespaceGroups) == 0 {
			return true, fmt.Sprintf("ignored all alerts in cluster '%s'", cluster)
		}

		// 3. Check NamespaceGroups trong cluster
		for _, ng := range c.NamespaceGroups {
			for _, nsPattern := range ng.Namespaces {
				if matchWithWildcard(nsPattern, namespace) && isWithinTimeRange(ng.Time) {
					return true, fmt.Sprintf("ignored by namespaceGroup %s in cluster %s", ng.Name, cluster)
				}
			}
		}

		// 4. Check namespace cụ thể
		for _, ns := range c.Namespaces {
			if !matchWithWildcard(ns.Name, namespace) {
				continue
			}

			// Nếu namespace còn trong time và không có pods
			if isWithinTimeRange(ns.Time) && len(ns.Pods) == 0 && len(ns.PodGroups) == 0 {
				return true, fmt.Sprintf("ignored all alerts in namespace '%s' with location '%s'", namespace, cluster)
			}

			// 5. Check PodGroups trong namespace
			for _, pg := range ns.PodGroups {
				for _, podPattern := range pg.Pods {
					if matchWithWildcard(podPattern, pod) && isWithinTimeRange(pg.Time) {
						return true, fmt.Sprintf("ignored by podGroup %s in %s/%s", pg.Name, cluster, namespace)
					}
				}
			}

			// 6. Check pod cụ thể
			for _, p := range ns.Pods {
				if matchWithWildcard(p.Name, pod) && isWithinTimeRange(p.Time) {
					return true, fmt.Sprintf("ignored pod '%s' with location '%s/%s'", pod, cluster, namespace)
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

		logEntry := fmt.Sprintf("[%s] Received alert:\n%s\n\n", nowLocal().Format(time.RFC3339), string(body))
		if _, err := logFile.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to write to alert log: %v\n", err)
		}

		// ---- TH1: Format K8S ----
		var alertData AlertData
		if err := json.Unmarshal(body, &alertData); err == nil && len(alertData.Alerts) > 0 {
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
			return
		}

		// ---- TH2: Format VM ----
		var vmAlert VMAlert
		if err := json.Unmarshal(body, &vmAlert); err == nil && vmAlert.State != "" {
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

			// Gửi đến đúng receiver
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
			return
		}

		http.Error(w, "invalid alert format", http.StatusBadRequest)
	}
}

func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
