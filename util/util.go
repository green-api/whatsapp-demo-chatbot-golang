package util

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"time"
)

func ContainString(optionVotes []string, targetWid string) bool {
	for _, voter := range optionVotes {
		if voter == targetWid {
			return true
		}
	}
	return false
}

func SessionCheck(notification chatbot.Notification) {
	lastTouchTime, ok := notification.GetStateData()["last_touch_timestamp"].(time.Time)
	if !ok {
		notification.UpdateStateData(map[string]interface{}{"last_touch_timestamp": time.Now()})
	}

	if !lastTouchTime.IsZero() && time.Since(lastTouchTime).Minutes() > 2 {
		notification.ActivateNextScene(notification.GetStartScene())
	} else {
		notification.UpdateStateData(map[string]interface{}{"last_touch_timestamp": time.Now()})
	}
}
