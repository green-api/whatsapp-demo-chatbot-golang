package util

import (
	"strings"
	"time"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/joho/godotenv"
)

func ContainString(optionVotes []string, targetWid string) bool {
	for _, voter := range optionVotes {
		if voter == targetWid {
			return true
		}
	}
	return false
}

func IsSessionExpired(notification *chatbot.Notification) bool {
	lastTouchTime, ok := notification.GetStateData()["last_touch_timestamp"].(time.Time)

	defer notification.UpdateStateData(map[string]interface{}{
		"last_touch_timestamp": time.Now(),
	})

	if !ok {
		return false
	}

	const sessionTimeout = 300.0

	if time.Since(lastTouchTime).Seconds() > sessionTimeout {
		return true
	}

	return false
}

func LinkPreview() string {
	envFile, err := godotenv.Read(".env")
	if err == nil {
		if val, exists := envFile["LINK_PREVIEW"]; exists && len(val) > 0 {
			if strings.ToLower(val) == "false" {
				return "false"
			}
		}
	}
	return "true"
}
