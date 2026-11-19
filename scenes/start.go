package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type StartScene struct{ Calls *CallHandlers }

func (s StartScene) Start(bot *chatbot.Bot) {
	s.Calls.register(bot, func(message *chatbot.Notification) {
		if util.IsSessionExpired(message) {
			message.SendText(util.GetString([]string{"select_language"}))
			message.ActivateNextScene(MainMenuScene{Calls: s.Calls})
		} else {
			// This code will be executed after idle timeout (10 minutes)
			//  from endpoints.go or mainMenu.go.
			// The start message is already printed from one of the file above
			//  and we are here from activating start scene.
			// The problem is: we should get to main menu state immediately.
			// That is why we calling SendMainMenu here.
			scene := MainMenuScene{Calls: s.Calls}
			scene.SendMainMenu(message)
		}
	})
}
