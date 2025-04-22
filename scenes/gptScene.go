package scenes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	gptbot "github.com/green-api/whatsapp-chatgpt-go"
	"github.com/green-api/whatsapp-demo-chatbot-golang/registry"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"github.com/sashabaranov/go-openai"
)

type GptScene struct{}

func (s GptScene) Start(bot *chatbot.Bot) {

	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if util.IsSessionExpired(message) {
			message.ActivateNextScene(StartScene{})
			message.SendText(util.GetString([]string{"select_language"}))
			return
		}

		lang := message.GetStateData()["lang"].(string)
		text, err := message.Text()
		if err != nil {
			text = ""
		}
		textLower := strings.ToLower(text)

		exitCommands := map[string][]string{
			"ru": {"меню", "выход", "стоп", "назад"},
			"en": {"menu", "exit", "stop", "back"},
			"he": {"תפריט", "יציאה", "עצור", "חזור", "exit"},
			"es": {"menú", "salir", "parar", "atrás", "exit"},
			"kz": {"мәзір", "шығу", "тоқта", "артқа", "меню"},
		}

		isExitCommand := false
		if commands, ok := exitCommands[lang]; ok {
			for _, cmd := range commands {
				if textLower == cmd {
					isExitCommand = true
					break
				}
			}
		}
		if textLower == "menu" {
			isExitCommand = true
		}

		if isExitCommand {
			var welcomeFileURL string
			if lang == "en" || lang == "es" || lang == "he" {
				welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_en.jpg"
			} else {
				welcomeFileURL = "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/welcome_ru.jpg"
			}
			senderName := ""
			if sd, ok := message.Body["senderData"].(map[string]interface{}); ok {
				if sn, ok := sd["senderName"].(string); ok {
					senderName = sn
				}
			}
			message.SendUrlFile(welcomeFileURL,
				"welcome.jpg",
				util.GetString([]string{"welcome_message", lang})+"*"+senderName+"*!"+"\n"+util.GetString([]string{"menu", lang}))

			message.ActivateNextScene(EndpointsScene{})
			return
		}

		ctx := context.Background()
		currentGptSessionData, err := loadGptSessionFromState(message)
		if err != nil {
			log.Printf("Error loading GPT session for %s, re-initializing: %v", message.StateId, err)
			currentGptSessionData = initializeGptSessionInState(message)
		}

		gptHelper := registry.GetGptHelper()
		if gptHelper == nil {
			log.Println("ERROR: GPT Helper not registered!")
			message.SendText(util.GetString([]string{"chat_gpt_error", lang}))
			return
		}

		response, updatedGptSessionData, err := gptHelper.ProcessMessage(ctx, message, currentGptSessionData)

		if err != nil {
			log.Printf("Error processing GPT message for %s: %v", message.StateId, err)
			message.SendText(util.GetString([]string{"chat_gpt_error", lang}))
			return
		}

		message.SendText(response)

		err = saveGptSessionToState(message, updatedGptSessionData) // Use renamed helper
		if err != nil {
			log.Printf("Error saving updated GPT session for %s: %v", message.StateId, err)
		}
	})
}

func initializeGptSessionInState(notification *chatbot.Notification) *gptbot.GPTSessionData {
	gptHelper := registry.GetGptHelper()
	if gptHelper == nil {
		log.Println("Error: Cannot initialize GPT session, helper bot not registered")
		return &gptbot.GPTSessionData{
			Messages:     []openai.ChatCompletionMessage{},
			LastActivity: time.Now().Unix(),
			UserData:     make(map[string]interface{}),
			Context:      make(map[string]interface{}),
		}
	}

	newSession := gptbot.GPTSessionData{
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: gptHelper.GetSystemMessage()},
		},
		LastActivity: time.Now().Unix(),
		UserData:     make(map[string]interface{}),
		Context:      make(map[string]interface{}),
	}

	err := saveGptSessionToState(notification, &newSession)
	if err != nil {
		log.Printf("Error saving initial GPT session for %s: %v", notification.StateId, err)
	}
	return &newSession
}

func loadGptSessionFromState(notification *chatbot.Notification) (*gptbot.GPTSessionData, error) {
	stateData := notification.GetStateData()
	if stateData == nil {
		return nil, fmt.Errorf("state data is nil")
	}

	jsonString, ok := stateData["gptSessionJson"].(string)
	if !ok || jsonString == "" {
		return nil, fmt.Errorf("gptSessionJson key not found or empty in state data")
	}

	var sessionData gptbot.GPTSessionData
	err := json.Unmarshal([]byte(jsonString), &sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal gptSessionJson from state: %w", err)
	}

	return &sessionData, nil
}

func saveGptSessionToState(notification *chatbot.Notification, sessionData *gptbot.GPTSessionData) error {
	if sessionData == nil {
		return fmt.Errorf("cannot save nil GPTSessionData")
	}

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data to JSON: %w", err)
	}

	stateData := notification.GetStateData()
	if stateData == nil {
		stateData = make(map[string]interface{})
	}

	stateData["gptSessionJson"] = string(jsonData)
	delete(stateData, "gptSession")

	notification.SetStateData(stateData)
	return nil
}
