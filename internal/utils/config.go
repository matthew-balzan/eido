package utils

import (
	"log"
	"os"

	"github.com/spf13/viper"

	"github.com/matthew-balzan/eido/internal/models"
	"github.com/matthew-balzan/eido/internal/vars"
)

func LoadConfig() (config models.Config, err error) {

	var env = "prod"
	var envCommand = os.Getenv("APP_ENV")

	if envCommand == "prod" || envCommand == "dev" {
		env = envCommand
	}

	viper.AddConfigPath("../../")
	viper.SetConfigFile("config-" + env + ".env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)

	// set config as global variable
	vars.Config = &config

	log.Println("Configs loaded!")
	return
}
