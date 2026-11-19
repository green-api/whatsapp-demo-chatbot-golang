package scenes

import (
	"encoding/json"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type CreateGroupScene struct{ Calls *CallHandlers }

func (s CreateGroupScene) Start(bot *chatbot.Bot) {

	s.Calls.register(bot, func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
			lang := message.GetStateData()["lang"].(string)
			text, _ := message.Text()
			senderId, _ := message.Sender()

			switch text {
			case "1":
				s.createGroup(message, lang, senderId)

			case "menu", "меню", "Menu", "Меню", "0":
				var welcomeFileURL string

				if lang == "en" {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_en.jpg"
				} else {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_ru.jpg"
				}

				message.SendUrlFile(welcomeFileURL, "welcome.jpg", util.GetString([]string{"menu", lang}))
				bot.ActivateNextScene(message.StateId, EndpointsScene{Calls: s.Calls})

			default:
				message.SendText(util.GetString([]string{"not_recognized_message", lang}))
			}
		} else {
			message.ActivateNextScene(MainMenuScene{Calls: s.Calls})
			message.SendText(util.GetString([]string{"select_language"}))
		}
	})
}

func (s CreateGroupScene) createGroup(message *chatbot.Notification, lang, senderId string) {
	groupResp, err := message.Groups().CreateGroup(
		util.GetString([]string{"group_name", lang}),
		[]string{senderId})

	if err != nil {
		*message.ErrorChannel <- err
		message.ActivateNextScene(EndpointsScene{Calls: s.Calls})
		return
	}

	var group map[string]interface{}
	_ = json.Unmarshal(groupResp.Body, &group)

	var groupId = group["chatId"].(string)
	message.StateManager.Create(groupId)
	message.StateManager.UpdateStateData(groupId, message.GetStateData())
	message.StateManager.ActivateNextScene(groupId, EndpointsScene{Calls: s.Calls})

	resp, err := message.Groups().SetGroupPicture(
		"assets/Group_avatar.jpg",
		groupId)

	if err != nil {
		*message.ErrorChannel <- err
	} else {
		var picResp map[string]interface{}
		_ = json.Unmarshal(resp.Body, &picResp)

		if picResp["setGroupPicture"].(bool) {
			_, err := message.Sending().SendMessage(
				groupId,
				util.GetString([]string{"send_group_message", lang})+util.GetString([]string{"links", lang, "groups_documentation"}))

			if err != nil {
				*message.ErrorChannel <- err
			}
		} else {
			_, err := message.Sending().SendMessage(
				groupId,
				util.GetString([]string{"send_group_message_set_picture_false", lang})+util.GetString([]string{"links", lang, "groups_documentation"}))

			if err != nil {
				*message.ErrorChannel <- err
			}
		}
	}

	message.SendText(util.GetString([]string{"group_created_message", lang}) +
		group["groupInviteLink"].(string))
	message.ActivateNextScene(EndpointsScene{Calls: s.Calls})
}
