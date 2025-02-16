package util

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"time"
	"strings"
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

	if ok && time.Since(lastTouchTime).Minutes() > 5 {
		notification.UpdateStateData(map[string]interface{}{"last_touch_timestamp": time.Now()})
		return true
	}

	notification.UpdateStateData(map[string]interface{}{"last_touch_timestamp": time.Now()})
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