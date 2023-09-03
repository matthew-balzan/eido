package handlers

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/matthew-balzan/eido/internal/commands"
	"github.com/matthew-balzan/eido/internal/vars"
)

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ignore messages by the bot
	if i.Member.User.ID == s.State.User.ID {
		return
	}

	instance := vars.Instances[i.GuildID]
	if instance == nil {
		vars.Instances[i.GuildID] = commands.CreateServerInstance(i.GuildID)
		instance = vars.Instances[i.GuildID]
	}

	middlewareLogger(s, i)

	// Check the interaction type
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		// Handle the slash command
		switch i.ApplicationCommandData().Name {
		case "ping":
			commands.PingCommand(s, i)
		// case "bet":
		// 	commands.BetCommand(s, i)
		case "play":
			commands.PlayCommand(s, i, instance)
		case "dc":
			commands.Disconnect(s, i, instance)
		case "endsong":
			commands.EndSong(s, i, instance)
		}
	}
}

func middlewareLogger(s *discordgo.Session, i *discordgo.InteractionCreate) {
	username := i.Member.User.Username
	command := i.ApplicationCommandData().Name
	log.Println(username + " used: " + command)
}
