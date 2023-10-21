package main

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/matthew-balzan/eido/internal/bot"
	"github.com/matthew-balzan/eido/internal/utils"
	"github.com/matthew-balzan/eido/internal/vars"
)

func main() {

	// Load configs
	_, err := utils.LoadConfig()

	if err != nil {
		log.Fatal("Cannot load config:", err)
	}

	// Create the session
	dg, err := discordgo.New(vars.Config.DiscordToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %s", err)
		return
	}

	// Initialize the bot
	bot := bot.NewBot(dg)

	// Register
	bot.RegisterCommands(dg)
	bot.RegisterHandlers()

	// Open connection
	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %s", err)
		return
	}

	log.Println("Running!")

	bot.WaitForTermination()
}
