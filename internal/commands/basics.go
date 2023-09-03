package commands

import (
	"github.com/bwmarrin/discordgo"
)

func SendSimpleMessageResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func SendSimpleMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.ChannelMessageSend(i.Interaction.ChannelID, message)
}

func SendComplexMessageResponse(s *discordgo.Session, i *discordgo.InteractionCreate, title string, description string, urlImage string, footerText string, color int) {

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
					Footer: &discordgo.MessageEmbedFooter{
						Text: footerText,
					},
					Image: &discordgo.MessageEmbedImage{
						URL: urlImage,
					},
					Author: &discordgo.MessageEmbedAuthor{
						Name: "Now playing:",
					},
				},
			},
		},
	})
}
