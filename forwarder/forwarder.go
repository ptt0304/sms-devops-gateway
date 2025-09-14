package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// const smsURL = "http://10.32.46.15:8082/sms/sendNumber"
const smsURL = "https://webhook.site/d3e6cee3-28a3-4153-8b20-dffe7894b787"

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

// package forwarder

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strings"
// )

// const smsURL = "https://webhook.site/5ca2dad3-7531-4d45-94e0-bf7faa5e1075"

// type SMSPayload struct {
// 	Mobile  string `json:"mobile"`
// 	Content string `json:"content"`
// }

// // SendSMS sends a message to a single mobile number
// func SendSMS(mobile string, message string) error {
// 	payload := SMSPayload{
// 		Mobile:  mobile,
// 		Content: message,
// 	}

// 	// In ra giống alert.go
// 	fmt.Println("📤 Sending SMS:")
// 	fmt.Printf("  Mobile : %s\n", payload.Mobile)
// 	fmt.Printf("  Content: %s\n", payload.Content)

// 	data, err := json.Marshal(payload)
// 	if err != nil {
// 		return fmt.Errorf("❌ Failed to marshal JSON: %v", err)
// 	}

// 	resp, err := http.Post(smsURL, "application/json", bytes.NewBuffer(data))
// 	if err != nil {
// 		return fmt.Errorf("❌ Failed to send SMS: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	bodyResp, _ := io.ReadAll(resp.Body)
// 	fmt.Printf("✅ Response status: %s. Body: %s\n", resp.Status, string(bodyResp))
// 	return nil
// }

// // SendToMultipleMobiles sends the same message to multiple mobile numbers
// func SendToMultipleMobiles(mobiles []string, message string) {
// 	for _, mobile := range mobiles {
// 		trimmed := strings.TrimSpace(mobile)
// 		if trimmed == "" {
// 			continue
// 		}
// 		if err := SendSMS(trimmed, message); err != nil {
// 			fmt.Printf("⚠️  Error sending SMS to %s: %v\n", trimmed, err)
// 		}
// 	}
// }
