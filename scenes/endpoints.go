package scenes

import (
	"encoding/json"
	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/model"
	"github.com/green-api/whatsapp-demo-chatbot-golang/registry"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"log"
	"strconv"
	"strings"
)

type EndpointsScene struct{}

func (s EndpointsScene) Start(bot *chatbot.Bot) {

	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
			lang := message.GetStateData()["lang"].(string)
			text, _ := message.Text()
			senderName := ""
			if sd, ok := message.Body["senderData"].(map[string]interface{}); ok {
				if sn, ok := sd["senderName"].(string); ok {
					senderName = sn
				}
			}
			senderId, _ := message.Sender()
			botNumber := ""
			if id, ok := message.Body["instanceData"].(map[string]interface{}); ok {
				if wid, ok := id["wid"].(string); ok {
					botNumber = wid
				}
			}

			if message.Filter(map[string][]string{"messageType": {"pollUpdateMessage"}}) {
				s.processPollUpdate(message, lang, senderId)
				return
			}

			switch text {
			case "1":
				message.SendText(util.GetString([]string{"send_text_message", lang})+util.GetString([]string{"links", lang, "send_text_documentation"}), "true")

			case "2":
				message.SendUrlFile(
					"https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/corgi.pdf",
					"corgi.pdf",
					util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

			case "3":
				message.SendUrlFile(
					"https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/corgi.jpg",
					"corgi.jpg",
					util.GetString([]string{"send_image_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

			case "4":
				message.SendText(util.GetString([]string{"send_audio_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}), "true")
				var fileLink = "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/Audio_bot_eng.mp3"
				if lang == "ru" {
					fileLink = "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/Audio_bot.mp3"
				}
				message.SendUrlFile(fileLink, "audio.mp3", "")

			case "5":
				var fileLink = "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/Video_bot_eng.mp4"
				if lang == "ru" {
					fileLink = "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/Video_bot_ru.mp4"
				}
				message.SendUrlFile(fileLink, "video.mp4",
					util.GetString([]string{"send_video_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))

			case "6":
				message.SendText(util.GetString([]string{"send_contact_message", lang})+util.GetString([]string{"links", lang, "send_contact_documentation"}), "true")
				phoneStrSender := strings.ReplaceAll(senderId, "@c.us", "")
				phoneIntSender, _ := strconv.Atoi(phoneStrSender)
				message.SendContact(greenapi.Contact{PhoneContact: phoneIntSender, FirstName: senderName})

			case "7":
				message.SendText(util.GetString([]string{"send_location_message", lang})+util.GetString([]string{"links", lang, "send_location_documentation"}), "true")
				message.SendLocation("", "", 35.888171, 14.440230)

			case "8":
				message.SendText(util.GetString([]string{"send_poll_message", lang})+
					util.GetString([]string{"links", lang, "send_poll_as_buttons"})+
					util.GetString([]string{"send_poll_message_1", lang})+
					util.GetString([]string{"links", lang, "send_poll_documentation"}), "true")

				message.SendPoll(util.GetString([]string{"poll_question", lang}), false,
					[]string{
						util.GetString([]string{"poll_option_1", lang}),
						util.GetString([]string{"poll_option_2", lang}),
						util.GetString([]string{"poll_option_3", lang}),
					})

			case "9":
				message.SendText(util.GetString([]string{"get_avatar_message", lang})+util.GetString([]string{"links", lang, "get_avatar_documentation"}), "true")
				resp, _ := message.Service().GetAvatar(senderId)
				var avatar map[string]interface{}
				_ = json.Unmarshal(resp.Body, &avatar)

				if avatarURL, ok := avatar["urlAvatar"].(string); ok && avatarURL != "" {
					message.SendUrlFile(
						avatarURL,
						"avatar.jpg",
						util.GetString([]string{"avatar_found", lang}))
				} else {
					message.SendText(util.GetString([]string{"avatar_not_found", lang}))
				}

			case "10":
				message.SendText(util.GetString([]string{"send_link_message_preview", lang})+util.GetString([]string{"links", lang, "send_link_documentation"}), "true")
				message.SendText(util.GetString([]string{"send_link_message_no_preview", lang})+util.GetString([]string{"links", lang, "send_link_documentation"}), "false")

			case "11":
				message.SendText(util.GetString([]string{"add_to_contact", lang}), "true")
				botPhoneStr := strings.ReplaceAll(botNumber, "@c.us", "")
				botPhoneInt, _ := strconv.Atoi(botPhoneStr)
				message.SendContact(greenapi.Contact{PhoneContact: botPhoneInt, FirstName: util.GetString([]string{"bot_name", lang})})
				message.ActivateNextScene(CreateGroupScene{})

			case "12":
				message.AnswerWithText(util.GetString([]string{"send_quoted_message", lang})+util.GetString([]string{"links", lang, "send_quoted_message_documentation"}), "true")

			case "13":
				message.SendUrlFile("https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/about_go.jpg", "logo.jpg",
					util.GetString([]string{"about_go_chatbot", lang})+
						util.GetString([]string{"link_to_docs", lang})+
						util.GetString([]string{"links", lang, "chatbot_documentation"})+
						util.GetString([]string{"link_to_source_code", lang})+
						util.GetString([]string{"links", lang, "chatbot_source_code"})+
						util.GetString([]string{"link_to_green_api", lang})+
						util.GetString([]string{"links", lang, "greenapi_website"})+
						util.GetString([]string{"link_to_console", lang})+
						util.GetString([]string{"links", lang, "greenapi_console"})+
						util.GetString([]string{"link_to_youtube", lang})+
						util.GetString([]string{"links", lang, "youtube_channel"}))

			case "14":
				gptHelper := registry.GetGptHelper()
				if gptHelper == nil {
					log.Println("Error: gptHelperBot is nil when trying to start GPT scene")
					message.SendText(util.GetString([]string{"sorry_message", lang}))
					return
				}

				message.SendText(util.GetString([]string{"chat_gpt_intro", lang}))

				_ = initializeGptSessionInState(message)

				message.ActivateNextScene(GptScene{})

			case "стоп", "Стоп", "stop", "Stop", "0":
				message.SendText(util.GetString([]string{"stop_message", lang})+"*"+senderName+"*!", "true")
				message.ActivateNextScene(StartScene{})

			case "menu", "меню", "Menu", "Меню":
				var welcomeFileURL string
				if lang == "en" || lang == "es" || lang == "he" {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_en.jpg"
				} else {
					welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_ru.jpg"
				}
				message.SendUrlFile(welcomeFileURL, "welcome.jpg", util.GetString([]string{"menu", lang}))
			case "":
			default:
				message.SendText(util.GetString([]string{"not_recognized_message", lang}), "true")
			}
		} else {
			message.ActivateNextScene(StartScene{})
			message.SendText(util.GetString([]string{"select_language"}))
		}
	})
}

func (s EndpointsScene) processPollUpdate(message *chatbot.Notification, lang string, senderId string) {
	webhookBody, _ := json.Marshal(message.Body)
	var pollMessage model.PollMessage
	if err := json.Unmarshal(webhookBody, &pollMessage); err != nil {
		log.Printf("Error unmarshalling poll update: %v", err)
		return
	}

	if pollMessage.MessageData.PollMessageData.Votes == nil || len(pollMessage.MessageData.PollMessageData.Votes) < 3 {
		log.Printf("Received poll update with unexpected vote structure for %s", message.StateId)
		return
	}

	isYes := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[0].OptionVoters, senderId)
	isNo := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[1].OptionVoters, senderId)
	isNothing := util.ContainString(pollMessage.MessageData.PollMessageData.Votes[2].OptionVoters, senderId)

	var messageText string
	if isYes {
		messageText = util.GetString([]string{"poll_answer_1", lang})
	} else if isNo {
		messageText = util.GetString([]string{"poll_answer_2", lang})
	} else if isNothing {
		messageText = util.GetString([]string{"poll_answer_3", lang})
	} else {
		return
	}

	message.SendText(messageText)
}
