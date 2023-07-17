package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/matthew-balzan/eido/internal/models"
)

func PlayCommand(s *discordgo.Session, i *discordgo.InteractionCreate, instance *models.ServerInstance) {
	if instance.Voice.IsPlaying {
		return
	}

	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	urlVideo := optionMap["url"].StringValue()

	channelId := ""

	channels := s.State.Guilds

	for _, c := range channels {
		voiceStates := c.VoiceStates
		for _, v := range voiceStates {
			if v.UserID == i.Member.User.ID {
				channelId = v.ChannelID
			}
		}
	}

	if channelId == "" {
		return
	}

	voiceConnection, err := s.ChannelVoiceJoin(i.GuildID, channelId, false, false)
	if err != nil {
		fmt.Println("ERR: internal/commands/audio.go: Error joining voice channel - ", err)
		return
	}

	instance.Voice.ChannelId = channelId
	instance.Voice.Connection = voiceConnection

	instance.Voice.PlaySingleSong(urlVideo)

}
