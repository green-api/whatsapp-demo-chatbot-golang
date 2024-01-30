package util

import (
	"fmt"
	envFile "github.com/joho/godotenv"
	config "github.com/realbucksavage/spring-config-client-go/v2"
	"os"
)

type ApplicationConfig struct {
	InstanceId string `json:"sapi_user_id" yaml:"sapi_user_id"`
	Token      string `json:"sapi_user_token" yaml:"sapi_user_token"`
	Link1      string `json:"slink_1" yaml:"slink_1"`
	Link2      string `json:"slink_2" yaml:"slink_2"`
}

func GetConfig() ApplicationConfig {
	if err := envFile.Load(); err != nil {
		fmt.Printf("Error loading .env file: %s\n", err)
	}

	client, err := config.NewClient(
		os.Getenv("SPRING_CLOUD_CONFIG_URI"),
		"sw-chatbot-go-7103",
		os.Getenv("ACTIVE_PROFILE"))
	if err != nil {
		panic(err)
	}

	var appConfig ApplicationConfig
	err = client.Decode(&appConfig)

	if err != nil {
		panic(err)
	}

	return appConfig
}
