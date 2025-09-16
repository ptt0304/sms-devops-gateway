package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// const smsURL = "http://10.32.46.15:8082/sms/sendNumber"
const smsURL = "https://webhook.site/490a0321-d6c2-4419-aa68-b5b05f32afe6"

type SMSPayload struct {
	Mobile  string `json:"mobile"`
	Content string `json:"content"`
}

// SendSMS sends a message to a single mobile number
func SendSMS(mobile string, message string) error {
	payload := SMSPayload{
		Mobile:  mobile,
		Content: message,
	}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(smsURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send SMS: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("SMS sent to %s. Response: %s\n", mobile, resp.Status)
	return nil
}

// SendToMultipleMobiles sends the same message to multiple mobile numbers
func SendToMultipleMobiles(mobiles []string, message string) {
	for _, mobile := range mobiles {
		trimmed := strings.TrimSpace(mobile)
		if trimmed == "" {
			continue
		}
		if err := SendSMS(trimmed, message); err != nil {
			fmt.Printf("Error sending SMS to %s: %v\n", trimmed, err)
		}
	}
}