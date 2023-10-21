package commands

import (
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"github.com/matthew-balzan/eido/internal/models"
)

// getAudioChannel returns the channelId.
// Returns an empty string if not found
func getAudioChannel(s *discordgo.Session, i *discordgo.InteractionCreate) (audioChannel string) {
	audioChannel = ""

	channels := s.State.Guilds

	for _, c := range channels {
		voiceStates := c.VoiceStates
		for _, v := range voiceStates {
			if v.UserID == i.Member.User.ID {
				audioChannel = v.ChannelID
			}
		}
	}

	return audioChannel
}

// checkAudioBasicPrerequisites returns false if we don't have the requisites, true otherwise.
// If it returns false and `response` is set to true, it automatically writes the error back to the user
func checkAudioBasicPrerequisites(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, response bool) (res bool) {
	// if the user is not in a voice channel
	if channelId == "" {
		if response {
			SendSimpleMessageResponse(s, i, "You have to join a voice channel")
		}
		return false
	}
	// if the bot is in another channel
	if instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId {
		if response {
			SendSimpleMessageResponse(s, i, "I'm playing in another channel")
		}
		return
	}
	return true
}

// isBotInAChannel returns false if it's not, true otherwise.
// If it returns false and `response` is set to true, it automatically writes the error back to the user
func isBotInAChannel(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, response bool) (res bool) {
	// if the bot is not in a channel
	if instance.Voice.Connection == nil {
		if response {
			SendSimpleMessageResponse(s, i, "I'm not in a voice channel right now")
		}
		return false
	}
	return true
}

// isBotPlaying returns false if it's not, true otherwise.
// If it returns false and `response` is set to true, it automatically writes the error back to the user
func isBotPlaying(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, response bool) (res bool) {
	// if the bot is not playing a song
	if !instance.Voice.IsPlaying {
		if response {
			SendSimpleMessageResponse(s, i, "I'm not playing anything right now")
		}
		return false
	}
	return true
}

func PlayCommand(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	url := optionMap["url"].StringValue()

	channelId := getAudioChannel(s, i)

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	log.Println(url)

	if strings.Contains(url, "playlist") {
		playCommandPlaylist(s, i, instance, channelId, url)
	} else {
		playCommandVideo(s, i, instance, channelId, url)
	}
}

func playCommandVideo(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlVideo string) {
	client := youtube.Client{}

	videoInfo, err := client.GetVideo(urlVideo)

	if err != nil || videoInfo == nil {
		SendSimpleMessageResponse(
			s,
			i,
			"Couldn't fetch the video, check if the url is correct",
		)
		return
	}

	if instance.Voice.Connection == nil { // if there's already a voice connection
		instance.Voice.startAudioSession(s, i, channelId) //start a new session
	}

	song := Song{
		url:       urlVideo,
		videoInfo: videoInfo,
	}

	result := instance.Voice.addToQueue(song)

	if result {
		SendSimpleMessageResponse(
			s,
			i,
			"*"+song.videoInfo.Title+"* added to queue",
		)
	} else {
		SendSimpleMessageResponse(
			s,
			i,
			"Couldnt add song to queue. Check if you went over the queue limit ("+strconv.Itoa(models.MaxQueueLength)+")",
		)
	}
}

func playCommandPlaylist(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlPlaylist string) {
	client := youtube.Client{}

	playlistInfo, err := client.GetPlaylist(urlPlaylist)

	if err != nil || playlistInfo == nil {
		log.Println(err)
		SendSimpleMessageResponse(
			s,
			i,
			"Couldn't fetch playlist. Check if it's public.",
		)
		return
	}

	if instance.Voice.Connection == nil { // if there's already a voice connection
		instance.Voice.startAudioSession(s, i, channelId) //start a new session
	}

	SendSimpleMessageResponse(
		s,
		i,
		"Adding playlist to queue. It may take some time ...",
	)

	globalError := false

	for _, entry := range playlistInfo.Videos {
		video, err := client.VideoFromPlaylistEntry(entry)

		if err != nil {
			globalError = true
		}

		song := Song{
			url:       "https://www.youtube.com/watch?v=" + video.ID,
			videoInfo: video,
		}

		result := instance.Voice.addToQueue(song)

		if !result {
			globalError = true
		}
	}

	if globalError {
		SendSimpleMessage(
			s,
			i,
			"Playlist added to queue, but one or more videos have not been added due to some errors. Check if you went over the limit of the queue ("+strconv.Itoa(models.MaxQueueLength)+")",
		)
	} else {
		SendSimpleMessage(
			s,
			i,
			"Playlist added to queue",
		)
	}
}

func Disconnect(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	SendSimpleMessageResponse(s, i, "Disconnecting")

	instance.Voice.disconnect()
}

func SkipSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	if !isBotPlaying(s, i, instance, true) {
		return
	}

	instance.Voice.skip()

	SendSimpleMessageResponse(s, i, "Song has been skipped")
}

func PauseSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	if !isBotPlaying(s, i, instance, true) {
		return
	}

	instance.Voice.setPause(true)

	SendSimpleMessageResponse(s, i, "Song has been paused")
}

func ResumeSong(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	if !isBotPlaying(s, i, instance, true) {
		return
	}

	instance.Voice.setPause(false)

	SendSimpleMessageResponse(s, i, "Song has been resumed")
}

func ClearQueue(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	instance.Voice.clearQueue()

	SendSimpleMessageResponse(s, i, "Queue cleared")
}

func GetQueue(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	queue := instance.Voice.getQueueList()
	var result = ""

	if len(queue) == 0 {
		result = "Queue is empty"
	} else {
		for i, song := range queue {
			result += strconv.Itoa(i) + ". " + song.videoInfo.Title + " \n"
		}
	}

	SendSimpleMessageResponse(s, i, result)
}
