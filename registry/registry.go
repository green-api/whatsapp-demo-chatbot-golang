package registry

import gptbot "github.com/green-api/whatsapp-chatgpt-go"

var gptHelperInstance *gptbot.WhatsappGptBot

func RegisterGptHelper(instance *gptbot.WhatsappGptBot) {
	gptHelperInstance = instance
}

func GetGptHelper() *gptbot.WhatsappGptBot {
	return gptHelperInstance
}
