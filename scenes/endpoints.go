package scenes

import (
	"encoding/json"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/model"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"log"
	"strings"
)

type EndpointsScene struct {
}

func (s EndpointsScene) Start(bot *chatbot.Bot) {
	cloudConfig := util.GetConfig()

	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
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
					cloudConfig.Link1,
					"corgi.png",
					util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

			case "3":
				message.AnswerWithUrlFile(
					cloudConfig.Link2,
					"corgi.jpg",
					util.GetString([]string{"send_image_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

			case "4":
				message.AnswerWithUploadFile(
					"assets/Audio_for_bot.mp3",
					util.GetString([]string{"send_audio_message", lang})+util.GetString([]string{"links", lang, "send_upload_documentation"}))

			case "5":
				message.AnswerWithUploadFile(
					"assets/Video_for_bot.mp4",
					util.GetString([]string{"send_video_message", lang})+util.GetString([]string{"links", lang, "send_upload_documentation"}))

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
					"assets/Group_avatar_bot.png",
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

	var messageText string
	if isYes {
		messageText = util.GetString([]string{"poll_response", lang, "if_yes"})
	} else if isNo {
		messageText = util.GetString([]string{"poll_response", lang, "if_no"})
	} else if isNothing {
		messageText = util.GetString([]string{"poll_response", lang, "if_nothing"})
	}

	message.Methods().Sending().SendMessage(map[string]interface{}{
		"message": messageText,
		"chatId":  chatId,
	})
}
