package commands

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	youtubeV2 "github.com/kkdai/youtube/v2"
	"github.com/matthew-balzan/eido/internal/models"
	"golang.org/x/net/html"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
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
			SendSimpleMessageResponse(s, i, "You have to join a voice channel", models.ColorError)
		}
		return false
	}
	// if the bot is in another channel
	if instance.Voice.Connection != nil && instance.Voice.Connection.ChannelID != channelId {
		if response {
			SendSimpleMessageResponse(s, i, "I'm playing in another channel", models.ColorError)
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
			SendSimpleMessageResponse(s, i, "I'm not in a voice channel right now", models.ColorError)
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
			SendSimpleMessageResponse(s, i, "I'm not playing anything right now", models.ColorError)
		}
		return false
	}
	return true
}

func PlayCommand(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, configs *models.Config) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	input := optionMap["input"].StringValue()

	channelId := getAudioChannel(s, i)

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	switch {
	case strings.Contains(input, "/playlist?"):
		playCommandPlaylist(s, i, instance, channelId, input)
	case (strings.Contains(input, "youtube.com") || strings.Contains(input, "youtu.be")):
		playCommandVideo(s, i, instance, channelId, input)
	case strings.Contains(input, "spotify.com"):
		title := getVideoTitleFromSpotify(input)
		url := searchVideoUrl(title, configs.YoutubeKey)
		playCommandVideo(s, i, instance, channelId, url)
	default:
		url := searchVideoUrl(input, configs.YoutubeKey)
		playCommandVideo(s, i, instance, channelId, url)
	}
}

func playCommandVideo(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlVideo string) {
	client := youtubeV2.Client{}

	videoInfo, err := client.GetVideo(urlVideo)

	if err != nil || videoInfo == nil {
		SendSimpleMessageResponse(
			s,
			i,
			"Couldn't fetch the video, check if the url is correct",
			models.ColorError,
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
			models.ColorDefault,
		)
	} else {
		SendSimpleMessageResponse(
			s,
			i,
			"Couldnt add song to queue. Check if you went over the queue limit ("+strconv.Itoa(models.MaxQueueLength)+")",
			models.ColorError,
		)
	}
}

func playCommandPlaylist(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlPlaylist string) {
	client := youtubeV2.Client{}

	playlistInfo, err := client.GetPlaylist(urlPlaylist)

	if err != nil || playlistInfo == nil {
		log.Println(err)
		SendSimpleMessageResponse(
			s,
			i,
			"Couldn't fetch playlist. Check if it's public.",
			models.ColorError,
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
		models.ColorDefault,
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
			models.ColorError,
		)
	} else {
		SendSimpleMessage(
			s,
			i,
			"Playlist added to queue",
			models.ColorDefault,
		)
	}
}

func getVideoTitleFromSpotify(input string) (url string) {
	res, _ := http.Get(input)
	doc, _ := html.Parse(res.Body)

	title := doc.FirstChild.NextSibling.FirstChild.FirstChild.NextSibling.FirstChild.Data //TODO

	log.Println(title)

	title = strings.ReplaceAll(title, "- song by", "")
	title = strings.ReplaceAll(title, "| Spotify", "")

	return title
}

func searchVideoUrl(input string, key string) (url string) {

	service, err := youtube.NewService(context.Background(), option.WithAPIKey(key))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	// Make the API call to YouTube.
	call := service.Search.List([]string{"id", "snippet"}).
		Q(input).
		MaxResults(5)
	response, _ := call.Do()

	// Iterate through each item and add it to the array if it's a video.
	for _, item := range response.Items {
		if item.Id.Kind == "youtube#video" {
			// return the first video
			return "https://www.youtube.com/watch?v=" + item.Id.VideoId
		}
	}

	return ""
}

func Disconnect(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance) {
	channelId := getAudioChannel(s, i)

	if !isBotInAChannel(s, i, instance, true) {
		return
	}

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	SendSimpleMessageResponse(s, i, "Disconnecting", models.ColorDefault)

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

	SendSimpleMessageResponse(s, i, "Song has been skipped", models.ColorDefault)
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

	SendSimpleMessageResponse(s, i, "Song has been paused", models.ColorDefault)
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

	SendSimpleMessageResponse(s, i, "Song has been resumed", models.ColorDefault)
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

	SendSimpleMessageResponse(s, i, "Queue cleared", models.ColorDefault)
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
	var message = ""

	if len(queue) == 0 {
		message = "Queue is empty"
	} else {
		for i, song := range queue {
			row := strconv.Itoa(i) + ". " + song.videoInfo.Title
			if i == 0 {
				row += " -> Now playing"
			}
			message += row + " \n"
		}
	}

	SendSimpleMessageResponse(s, i, message, models.ColorDefault)
}
