package commands

import (
	"github.com/bwmarrin/discordgo"
)

func PingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {

	responseMessage := "Pong!"

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseMessage,
		},
	})
}
