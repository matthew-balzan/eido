package bot

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/matthew-balzan/eido/internal/handlers"
)

type Bot struct {
	session *discordgo.Session
}

func NewBot(session *discordgo.Session) *Bot {
	return &Bot{
		session: session,
	}
}

func (b *Bot) RegisterCommands(session *discordgo.Session) {

	// Register the slash commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong",
		},
		// {
		// 	Name:        "bet",
		// 	Description: "Create a bet",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "title",
		// 			Description: "Title of the bet",
		// 			Required:    true,
		// 		},
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "choice1",
		// 			Description: "Choice 1",
		// 			Required:    true,
		// 		},
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "choice2",
		// 			Description: "Choice 2",
		// 			Required:    true,
		// 		},
		// 	},
		// },
		{
			Name:        "play",
			Description: "Adds a song or playlist to the queue. Supports youtube links, spotify links and youtube search",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "input",
					Description: "Url of the song or search input",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "skip-playlist",
					Description: "Number of songs to skip in case the input is a playlist",
					Required:    false,
				},
			},
		},
		{
			Name:        "skip",
			Description: "Skips the current song",
		},
		{
			Name:        "pause",
			Description: "Pause the current song",
		},
		{
			Name:        "resume",
			Description: "Resume the current song",
		},
		{
			Name:        "disconnect",
			Description: "Disconnects the bot from the voice channel",
		},
		{
			Name:        "clear",
			Description: "Clear the queue",
		},
		{
			Name:        "queue",
			Description: "Lists the songs in the queue",
		},
	}

	app, err := session.Application("@me")
	if err != nil {
		log.Println("Error getting application information:", err)
		return
	}

	_, err = session.ApplicationCommandBulkOverwrite(app.ID, "", commands)
	if err != nil {
		log.Println("Error registering slash commands:", err)
		return
	}

	log.Println("Slash commands registered!")
}

func (b *Bot) RegisterHandlers() {
	b.session.AddHandler(handlers.InteractionCreate)
}

func (b *Bot) WaitForTermination() {
	// Wait here until terminaion
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.session.Close()
}
