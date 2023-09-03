package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"github.com/matthew-balzan/eido/internal/models"
)

func PlayCommand(s *discordgo.Session, i *discordgo.InteractionCreate, instance *models.ServerInstance) {
	if instance.Voice.IsPlaying {
		SendSimpleMessage(s, i, "I'm already playing a song")
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
		SendSimpleMessage(s, i, "You have to join a voice channel to play a song")
		return
	}

	voiceConnection, err := s.ChannelVoiceJoin(i.GuildID, channelId, false, false)
	if err != nil {
		fmt.Println("ERR: internal/commands/audio.go: Error joining voice channel - ", err)
		return
	}

	instance.Voice.ChannelId = channelId
	instance.Voice.Connection = voiceConnection

	client := youtube.Client{}

	videoInfo, err := client.GetVideo(urlVideo)
	if err != nil {
		fmt.Println("ERR: internal/models/instance.go: Couldn't fetch video info - ", err)
		return
	}

	instance.Voice.IsPlaying = true

	SendComplexMessage(s, i, videoInfo.Title, urlVideo, videoInfo.Thumbnails[1].URL, videoInfo.Duration.String())

	instance.Voice.PlaySingleSong(videoInfo)

	voiceConnection.Disconnect()

	instance.Voice.IsPlaying = false

}
