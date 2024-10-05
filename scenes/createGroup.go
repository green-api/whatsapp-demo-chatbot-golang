package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type CreateGroupScene struct {
}

func (s CreateGroupScene) Start(bot *chatbot.Bot) {

	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
			lang := message.GetStateData()["lang"].(string)
			text, _ := message.Text()
			senderId, _ := message.Sender()

			switch text {
			case "1":
				group, err := message.GreenAPI.Methods().Groups().CreateGroup(
					util.GetString([]string{"group_name", lang}),
					[]string{senderId})
				if err != nil {
					*message.ErrorChannel <- err
				}

				var groupId = group["chatId"].(string)
				message.StateManager.Create(groupId)
				message.StateManager.UpdateStateData(groupId, message.GetStateData())
				message.StateManager.ActivateNextScene(groupId, EndpointsScene{})

				resp, err := message.GreenAPI.Methods().Groups().SetGroupPicture(
					"assets/Group_avatar.jpg",
					groupId)
				if err != nil {
					*message.ErrorChannel <- err
				}

				if resp["setGroupPicture"].(bool) {
					_, err := message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
						"chatId":  groupId,
						"message": util.GetString([]string{"send_group_message", lang}) + util.GetString([]string{"links", lang, "groups_documentation"})})
					if err != nil {
						bot.ErrorChannel <- err
					}
				} else {
					_, err := message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
						"chatId":  groupId,
						"message": util.GetString([]string{"send_group_message_set_picture_false", lang}) + util.GetString([]string{"links", lang, "groups_documentation"})})
					if err != nil {
						bot.ErrorChannel <- err
					}
				}
				message.SendText(util.GetString([]string{"group_created_message", lang}) +
					group["groupInviteLink"].(string))
				message.ActivateNextScene(EndpointsScene{})

			case "menu", "меню", "Menu", "Меню", "0":
				var welcomeFileURL string
				if lang == "en" {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_en.jpg"
				} else {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_ru.jpg"
				}
				message.SendUrlFile(welcomeFileURL, "welcome.jpg", util.GetString([]string{"menu", lang}))
				bot.ActivateNextScene(message.StateId, EndpointsScene{})

			default:
				message.SendText(util.GetString([]string{"not_recognized_message", lang}))
			}
		} else {
			message.ActivateNextScene(MainMenuScene{})
			message.SendText(util.GetString([]string{"select_language"}))
		}
	})
}
