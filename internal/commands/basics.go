package commands

import (
	"github.com/bwmarrin/discordgo"
)

func chunks(s string, chunkSize int) []string {
	if len(s) == 0 {
		return []string{s}
	}
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string = make([]string, 0, (len(s)-1)/chunkSize+1)
	currentLen := 0
	currentStart := 0
	for i := range s {
		if currentLen == chunkSize {
			chunks = append(chunks, s[currentStart:i])
			currentLen = 0
			currentStart = i
		}
		currentLen++
	}
	chunks = append(chunks, s[currentStart:])
	return chunks
}

func SendSimpleMessageResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string, color int) {

	multipleMessages := chunks(message, 2000)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: multipleMessages[0],
					Color:       color,
				},
			},
		},
	})

	if len(multipleMessages) > 1 {
		for index := range multipleMessages {
			if index < 1 {
				continue
			}
			SendSimpleMessage(s, i, multipleMessages[index], color)
		}
	}
}

func SendSimpleMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string, color int) {
	multipleMessages := chunks(message, 2000)
	s.ChannelMessageSendEmbeds(i.ChannelID, []*discordgo.MessageEmbed{
		{
			Description: multipleMessages[0],
			Color:       color,
		},
	})

	if len(multipleMessages) > 1 {
		for index := range multipleMessages {
			if index < 1 {
				continue
			}
			SendSimpleMessage(s, i, multipleMessages[index], color)
		}
	}
}

func SendComplexMessageResponse(s *discordgo.Session, i *discordgo.InteractionCreate, title string, description string, urlImage string, footerText string, color int, author string) {
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
						Name: author,
					},
				},
			},
		},
	})
}

func SendComplexMessage(s *discordgo.Session, i *discordgo.InteractionCreate, title string, description string, urlImage string, footerText string, color int, author string) {

	s.ChannelMessageSendEmbeds(i.ChannelID, []*discordgo.MessageEmbed{
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
				Name: author,
			},
		},
	})
}
