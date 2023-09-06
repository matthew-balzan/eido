package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"github.com/matthew-balzan/eido/internal/models"
)

func PlayCommand(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	if channelId == "" { // if the user is not in a voice channel
		SendSimpleMessageResponse(s, i, "You have to join a voice channel to play a song")
		return
	}

	if instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId { // if the bot is in another channel
		SendSimpleMessageResponse(s, i, "I'm playing in another channel")
		return
	}

	if instance.Voice.IsPlaying { //if the bot is already playing a song
		SendSimpleMessageResponse(s, i, "I'm already playing a song, you have to wait or skip it")
		return
	}

	var voiceConnection *discordgo.VoiceConnection = nil

	if instance.Voice.Connection != nil { // if there's already a voice connection
		voiceConnection = instance.Voice.Connection // take that one
	} else { // else join the channel
		var err error = nil
		voiceConnection, err = s.ChannelVoiceJoin(i.GuildID, channelId, false, false)
		if err != nil {
			log.Println("ERR: internal/commands/audio.go: Error joining voice channel - ", err)
			return
		}
	}

	client := youtube.Client{}

	videoInfo, err := client.GetVideo(urlVideo)
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Couldn't fetch video info - ", err)
		return
	}

	SendComplexMessageResponse(
		s,
		i,
		videoInfo.Title,
		urlVideo,
		videoInfo.Thumbnails[1].URL,
		"Duration: "+videoInfo.Duration.String(),
		models.ColorRed,
	)

	instance.Voice.StopTimer()
	instance.Voice.IsPlaying = true
	instance.Voice.ChannelId = channelId
	instance.Voice.Connection = voiceConnection

	instance.Voice.PlaySingleSong(videoInfo)

	instance.Voice.IsPlaying = false

	instance.Voice.StartTimer(s, i)

}

func Disconnect(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	if instance.Voice.Connection == nil { // if the bot is not in a channel
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	if channelId == "" { // if the user is not in a voice channel
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId { // if the bot is in another channel
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	instance.Voice.Connection.Disconnect()
	instance.Voice.Connection = nil
	instance.Voice.ChannelId = ""
	instance.Voice.Timer.Stop()

	SendSimpleMessageResponse(s, i, "Disconnected")

}

func EndSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	if instance.Voice.Connection == nil { // if the bot is not in a channel
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	if channelId == "" { // if the user is not in a voice channel
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId { // if the bot is in another channel
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if !instance.Voice.IsPlaying { // if the bot is not playing a song
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	instance.Voice.Encoder.Stop()

	SendSimpleMessageResponse(s, i, "Song has ended")

	instance.Voice.IsPlaying = false

}
