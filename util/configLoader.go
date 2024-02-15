package util

import (
	"github.com/green-api/whatsapp-demo-chatbot-golang/config"
	"log"
	"os"
	"strings"
)

var CloudConfig config.Data

func GetConfig() {
	profile := config.ParseProfile(os.Getenv("ACTIVE_PROFILE"))

	cloudConfig := config.NewCloudConfig()

	log.Println("Loading cloud config")
	err := cloudConfig.Load(
		"sw-chatbot-go", profile.Pool, strings.Join(
			[]string{profile.Pool, profile.Server}, "",
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Getting cloud config")
	data, err := cloudConfig.Get("sw-chatbot-go-" + profile.Pool)
	if err != nil {
		log.Fatalln(err)
	}

	CloudConfig = *data
}
