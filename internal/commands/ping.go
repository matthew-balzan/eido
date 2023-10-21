package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/matthew-balzan/eido/internal/models"
)

func PingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	responseMessage := "Pong!"
	SendSimpleMessageResponse(s, i, responseMessage, models.ColorDefault)
}
