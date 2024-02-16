package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"strings"
)

type CreateGroupScene struct {
}

func (s CreateGroupScene) Start(bot *chatbot.Bot) {

	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
			lang := message.GetStateData()["lang"].(string)
			text, _ := message.Text()
			senderId, _ := message.Sender()
			botNumber, _ := message.Body["instanceData"].(map[string]interface{})["wid"].(string)

			switch text {
			case "1":
				group, err := message.GreenAPI.Methods().Groups().CreateGroup(
					util.GetString([]string{"group_name", lang}),
					[]string{senderId})
				if err != nil {
					*message.ErrorChannel <- err
				}

				resp, err := message.GreenAPI.Methods().Groups().SetGroupPicture(
					"assets/Group_avatar.jpg",
					group["chatId"].(string))
				if err != nil {
					*message.ErrorChannel <- err
				}

				if resp["setGroupPicture"].(bool) {
					_, err := message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
						"chatId":  group["chatId"].(string),
						"message": util.GetString([]string{"send_group_message", lang}) + util.GetString([]string{"links", lang, "groups_documentation"})})
					if err != nil {
						bot.ErrorChannel <- err
					}
				} else {
					_, err := message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
						"chatId":  group["chatId"].(string),
						"message": util.GetString([]string{"send_group_message_set_picture_false", lang}) + util.GetString([]string{"links", lang, "groups_documentation"})})
					if err != nil {
						bot.ErrorChannel <- err
					}
				}
				message.SendText(util.GetString([]string{"group_created_message", lang}) +
					group["groupInviteLink"].(string))
				message.ActivateNextScene(EndpointsScene{})
			case "0":
				message.SendText(util.GetString([]string{"menu", lang}))
				bot.ActivateNextScene(message.StateId, EndpointsScene{})
			case "menu", "меню", "Menu", "Меню":
				message.SendText(util.GetString([]string{"add_to_contact", lang}))
				message.SendContact(map[string]interface{}{"firstName": util.GetString([]string{"bot_name", lang}), "phoneContact": strings.ReplaceAll(botNumber, "@c.us", "")})
			default:
				message.SendText(util.GetString([]string{"not_recognized_message", lang}))
			}
		} else {
			message.ActivateNextScene(MainMenuScene{})
			message.SendText(util.GetString([]string{"select_language"}))
		}
	})
}
