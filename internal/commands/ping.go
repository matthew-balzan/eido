package commands

import (
	"github.com/bwmarrin/discordgo"
)

func PingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	responseMessage := "Pong!"
	SendSimpleMessageResponse(s, i, responseMessage)
}
