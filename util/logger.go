package util

import (
	"encoding/json"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type CustomFormatter struct {
	Location  *time.Location
	Project   string
	Service   string
	System    string
	Pool      string
	Server    string
	Instance  string
	Marker    string
	Container string
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields)
	for k, v := range entry.Data {
		data[k] = v
	}
	data["project"] = f.Project
	data["timestamp"] = entry.Time.In(f.Location).Format("02.01.2006, 15:04:05")
	data["level"] = entry.Level.String()
	data["service"] = f.Service
	data["system"] = f.System
	data["instance"] = f.Instance
	data["pool"] = f.Pool
	data["server"] = f.Server
	data["container"] = f.Container
	data["marker"] = f.Marker
	data["message"] = entry.Message
	line, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(line, '\n'), nil
}

var Logger *logrus.Logger

func GetLogger() *logrus.Logger {
	log := logrus.New()

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalln(err)
	}

	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&CustomFormatter{
		Location:  location,
		Project:   CloudConfig.Project,
		Service:   CloudConfig.Service,
		System:    CloudConfig.System,
		Container: CloudConfig.Container,
		Instance:  strconv.FormatInt(CloudConfig.InstanceId, 10),
	})

	Logger = log
	return log
}

func Log(message *chatbot.Notification, marker string) {
	chatId, err := message.ChatId()
	senderId, err := message.Sender()
	if err != nil {
		*message.ErrorChannel <- err
	}
	Logger.Debugln(marker, "messageId:", message.Body["idMessage"], "chatId:", chatId, "senderId:", senderId)
}
