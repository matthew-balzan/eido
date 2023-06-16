package handlers

import (
	"github.com/bwmarrin/discordgo"

	"github.com/matthew-balzan/eido/internal/commands"
)

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ignore messages by the bot
	if i.Member.User.ID == s.State.User.ID {
		return
	}

	// Check the interaction type
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		// Handle the slash command
		switch i.ApplicationCommandData().Name {
		case "ping":
			commands.PingCommand(s, i)
		case "bet":
			commands.BetCommand(s, i)
		}
	}
}
