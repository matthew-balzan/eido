package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
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

	if instance.Voice.Connection == nil { // if there's already a voice connection
		instance.Voice.startAudioSession(s, i, channelId)
	}

	client := youtube.Client{}

	videoInfo, err := client.GetVideo(urlVideo)
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Couldn't fetch video info - ", err)
		return
	}

	song := Song{
		url:       urlVideo,
		videoInfo: videoInfo,
	}

	SendSimpleMessageResponse(
		s,
		i,
		"*"+song.videoInfo.Title+"* added to queue",
	)

	instance.Voice.addToQueue(song)
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

	SendSimpleMessageResponse(s, i, "Disconnecting")

	instance.Voice.Connection.Disconnect()
	instance.Voice.Connection = nil
	instance.Voice.ChannelId = ""
	instance.Voice.Stream = nil
	instance.Voice.StopTimer()
	if instance.Voice.Queue != nil {
		close(instance.Voice.Queue)
	}
}

func SkipSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	// if the bot is not in a channel
	if instance.Voice.Connection == nil {
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	// if the user is not in a voice channel or not in the same channel
	if channelId == "" || (instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId) {
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if !instance.Voice.IsPlaying { // if the bot is not playing a song
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	instance.Voice.skip()

	SendSimpleMessageResponse(s, i, "Song has been skipped")
}

func PauseSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	// if the bot is not in a channel
	if instance.Voice.Connection == nil {
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	// if the user is not in a voice channel or not in the same channel
	if channelId == "" || (instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId) {
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if !instance.Voice.IsPlaying { // if the bot is not playing a song
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	instance.Voice.setPause(true)

	SendSimpleMessageResponse(s, i, "Song has been paused")
}

func ResumeSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
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

	// if the bot is not in a channel
	if instance.Voice.Connection == nil {
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	// if the user is not in a voice channel or not in the same channel
	if channelId == "" || (instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId) {
		SendSimpleMessageResponse(s, i, "You have to join the voice channel I'm at to disconnect me")
		return
	}

	if !instance.Voice.IsPlaying { // if the bot is not playing a song
		SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		return
	}

	instance.Voice.setPause(false)

	SendSimpleMessageResponse(s, i, "Song has been resumed")
}
