package util

import (
	"encoding/json"
	"github.com/green-api/whatsapp-demo-chatbot-golang/config"
	"github.com/sirupsen/logrus"
	"github.com/sohlich/elogrus"
	"gopkg.in/olivere/elastic.v5"
	"os"
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
	data["pool"] = f.Pool
	data["server"] = f.Server
	data["instance"] = strconv.FormatInt(CloudConfig.InstanceId, 10)
	data["container"] = f.Container
	data["message"] = entry.Message
	line, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(line, '\n'), nil
}

func GetLogger() *logrus.Logger {
	profile := config.ParseProfile(os.Getenv("ACTIVE_PROFILE"))
	log := logrus.New()

	client, err := elastic.NewClient(elastic.SetURL(CloudConfig.ElasticUrl))
	if err != nil {
		log.Fatal("Unable to create Elasticsearch client: ", err)
	}

	hook, err := elogrus.NewElasticHook(client, CloudConfig.ElasticHost, logrus.DebugLevel, "sw-chatbot-go")
	if err != nil {
		log.Error("Unable to create Elasticsearch hook: ", err)
	}
	log.Hooks.Add(hook)

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
		Pool:      profile.Pool,
		Server:    profile.Server,
		Instance:  strconv.FormatInt(CloudConfig.InstanceId, 10),
	})

	return log
}
