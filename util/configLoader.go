package util

import (
	"github.com/green-api/whatsapp-demo-chatbot-golang/config"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type Formatter struct {
	Location  *time.Location
	Formatter *logrus.JSONFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(f.Location)

	return f.Formatter.Format(entry)
}

func GetConfig() config.Data {
	log := logrus.New()

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalln(err)
	}

	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&Formatter{
		Location: location,
		Formatter: &logrus.JSONFormatter{
			TimestampFormat: time.DateTime,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:  "message",
				logrus.FieldKeyTime: "timestamp",
			},
		},
	})

	profile := config.ParseProfile(os.Getenv("ACTIVE_PROFILE"))

	cloudConfig := config.NewCloudConfig(log)

	log.Infoln("Loading cloud config")
	err = cloudConfig.Load(
		"sw-chatbot-go", profile.Pool, strings.Join(
			[]string{profile.Pool, profile.Server}, "",
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	log.Infoln("Getting cloud config")
	data, err := cloudConfig.Get("sw-chatbot-go-" + profile.Pool)
	if err != nil {
		log.Fatalln(err)
	}

	return *data
}
