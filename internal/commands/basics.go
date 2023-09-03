package commands

import (
	"github.com/bwmarrin/discordgo"
)

func SendSimpleMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func SendComplexMessage(s *discordgo.Session, i *discordgo.InteractionCreate, title string, description string, urlImage string, footerText string) {

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       15548997, //red
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
