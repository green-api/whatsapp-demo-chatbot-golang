package scenes

import (
	"demo-chatbot/model"
	"demo-chatbot/util"
	"encoding/json"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"log"
	"strings"
)

type EndpointsScene struct {
}

func (s EndpointsScene) Start(bot *chatbot.Bot) {
	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		lang := message.GetStateData()["lang"].(string)
		text, _ := message.Text()
		senderName := message.Body["senderData"].(map[string]interface{})["senderName"].(string)
		messageId := message.Body["idMessage"].(string)
		chatId, _ := message.ChatId()

		if message.Filter(map[string][]string{"messageType": {"pollUpdateMessage"}}) {
			s.processPollUpdate(message, chatId, lang)
		}

		switch text {
		case "1":
			message.AnswerWithText(util.GetString([]string{"send_text_message", lang}) + util.GetString([]string{"links", lang, "send_text_documentation"}))

		case "2":
			message.AnswerWithUrlFile(
				"https://images.rawpixel.com/image_png_1100/cHJpdmF0ZS9sci9pbWFnZXMvd2Vic2l0ZS8yMDIzLTExL3Jhd3BpeGVsb2ZmaWNlMTlfcGhvdG9fb2ZfY29yZ2lzX2luX2NocmlzdG1hc19zd2VhdGVyX2luX2FfcGFydF80YWM1ODk3Zi1mZDMwLTRhYTItYWM5NS05YjY3Yjg1MTFjZmUucG5n.png",
				"corgi.png",
				util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

		case "3":
			message.AnswerWithUrlFile(
				"https://images.rawpixel.com/image_png_1100/cHJpdmF0ZS9sci9pbWFnZXMvd2Vic2l0ZS8yMDIzLTExL3Jhd3BpeGVsb2ZmaWNlMTlfcGhvdG9fb2ZfY29yZ2lzX2luX2NocmlzdG1hc19zd2VhdGVyX2luX2FfcGFydF80YWM1ODk3Zi1mZDMwLTRhYTItYWM5NS05YjY3Yjg1MTFjZmUucG5n.png",
				"corgi.jpg",
				util.GetString([]string{"send_image_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

		case "4":

		case "5":

		case "6":
			message.AnswerWithText(util.GetString([]string{"send_contact_message", lang}) + util.GetString([]string{"links", lang, "send_contact_documentation"}))
			message.AnswerWithContact(map[string]interface{}{"firstName": senderName, "phoneContact": strings.ReplaceAll(chatId, "@c.us", "")})

		case "7":
			message.AnswerWithText(util.GetString([]string{"send_location_message", lang}) + util.GetString([]string{"links", lang, "send_location_documentation"}))
			message.AnswerWithLocation("", "", 35.888171, 14.440230)

		case "8":
			message.AnswerWithText(util.GetString([]string{"send_poll_message", lang}))
			message.AnswerWithPoll(util.GetString([]string{"poll_name", lang}), false,
				[]map[string]interface{}{
					{"optionName": util.GetString([]string{"poll_option", lang, "o1"})},
					{"optionName": util.GetString([]string{"poll_option", lang, "o2"})},
					{"optionName": util.GetString([]string{"poll_option", lang, "o3"})},
				})

		case "9":
			message.AnswerWithText(util.GetString([]string{"send_avatar_message", lang, "avatar"}))
			avatar, _ := message.GreenAPI.Methods().Service().GetAvatar(chatId)

			if avatar["urlAvatar"] != nil {
				message.AnswerWithUrlFile(
					avatar["urlAvatar"].(string),
					"avatar",
					util.GetString([]string{"send_avatar_message", lang, "avatar_exist"}))
			} else {
				message.AnswerWithText(util.GetString([]string{"send_avatar_message", lang, "avatar_not_exist"}))
			}

		case "10":
			message.AnswerWithText(util.GetString([]string{"send_link_message", lang, "with_preview"}))
			message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
				"chatId":          chatId,
				"message":         util.GetString([]string{"send_link_message", lang, "without_preview"}),
				"quotedMessageId": messageId,
				"linkPreview":     false,
			})

		case "11":
			group, _ := message.GreenAPI.Methods().Groups().CreateGroup(
				util.GetString([]string{"group_name", lang}),
				[]string{chatId})
			message.GreenAPI.Methods().Groups().SetGroupPicture(
				"src/main/resources/img.png",
				group["chatId"].(string))
			message.GreenAPI.Methods().Sending().SendMessage(map[string]interface{}{
				"chatId":  group["chatId"].(string),
				"message": util.GetString([]string{"create_group_message", lang}),
			})

		case "стоп", "Стоп", "stop", "Stop":
			message.AnswerWithText(util.GetString([]string{"stop_message", lang}) + senderName)
			message.ActivateNextScene(StartScene{})

		case "menu", "меню", "Menu", "Меню":
			message.AnswerWithText(util.GetString([]string{"menu", lang}))

		case "":
		default:
			message.AnswerWithText(util.GetString([]string{"not_recognized_message", lang}))
		}
	})
}

func (s EndpointsScene) processPollUpdate(message *chatbot.Notification, chatId string, lang string) {
	webhookBody, _ := json.Marshal(message.Body)
	var pollMessage model.PollMessage
	if err := json.Unmarshal(webhookBody, &pollMessage); err != nil {
		log.Fatal(err)
	}

	isYes := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[0].OptionVoters, chatId)
	isNo := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[1].OptionVoters, chatId)
	isNothing := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[2].OptionVoters, chatId)

	if isYes {
		message.AnswerWithText(util.GetString([]string{"poll_response", lang, "if_yes"}))
	} else if isNo {
		message.AnswerWithText(util.GetString([]string{"poll_response", lang, "if_no"}))
	} else if isNothing {
		message.AnswerWithText(util.GetString([]string{"poll_response", lang, "if_nothing"}))
	}
}
