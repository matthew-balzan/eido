package utils

import (
	"log"

	"github.com/spf13/viper"

	"github.com/matthew-balzan/eido/internal/models"
)

var Config *models.Config

func LoadConfig() (config models.Config, err error) {
	viper.AddConfigPath("../../")
	viper.SetConfigFile(".env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)

	// set config as global variable
	Config = &config

	log.Println("Configs loaded!")
	return
}