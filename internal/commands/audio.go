package commands

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
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
	var skip uint64 = 0
	if optionMap["skip-playlist"] != nil {
		skip = optionMap["skip-playlist"].UintValue()
	}

	channelId := getAudioChannel(s, i)

	if !checkAudioBasicPrerequisites(s, i, instance, channelId, true) {
		return
	}

	switch {
	case strings.Contains(input, "/playlist?"):
		playCommandPlaylist(s, i, instance, channelId, input, skip, configs.YoutubeKey)
	case (strings.Contains(input, "youtube.com") || strings.Contains(input, "youtu.be")):
		playCommandVideo(s, i, instance, channelId, input, configs.YoutubeKey)
	case strings.Contains(input, "spotify.com"):
		title := getVideoTitleFromSpotify(input)
		url := searchVideoUrl(title, configs.YoutubeKey)
		playCommandVideo(s, i, instance, channelId, url, configs.YoutubeKey)
	default:
		url := searchVideoUrl(input, configs.YoutubeKey)
		playCommandVideo(s, i, instance, channelId, url, configs.YoutubeKey)
	}
}

func playCommandVideo(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlVideo string, key string) {

	reg := `^.*(?:(?:youtu\.be\/|v\/|vi\/|u\/\w\/|embed\/|shorts\/)|(?:(?:watch)?\?v(?:i)?=|\&v(?:i)?=))([^#\&\?]*).*`
	res := regexp.MustCompile(reg)
	id := res.FindStringSubmatch(urlVideo)[1]

	service, err := youtube.NewService(context.Background(), option.WithAPIKey(key))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	call := service.Videos.List([]string{"contentDetails", "snippet"}).Id(id)
	resYt, _ := call.Do()

	var videoInfo VideoInfo
	if resYt.Items[0] != nil {
		videoInfo = VideoInfo{
			ID:        resYt.Items[0].Id,
			Title:     resYt.Items[0].Snippet.Title,
			Author:    resYt.Items[0].Snippet.ChannelTitle,
			Duration:  strings.ReplaceAll(resYt.Items[0].ContentDetails.Duration, "PT", ""),
			Thumbnail: resYt.Items[0].Snippet.Thumbnails.Default.Url,
		}
	}

	if err != nil || videoInfo.ID == "" {
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

func playCommandPlaylist(s *discordgo.Session, i *discordgo.InteractionCreate, instance *ServerInstance, channelId string, urlPlaylist string, skip uint64, key string) {

	list := []VideoInfo{}
	page := ""

	reg := `^.*?(?:v|list)=(.*?)(?:&|$)`
	res := regexp.MustCompile(reg)
	id := res.FindStringSubmatch(urlPlaylist)[1]

	service, err := youtube.NewService(context.Background(), option.WithAPIKey(key))
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	for cont := true; cont; {
		call := service.PlaylistItems.List([]string{"contentDetails", "snippet"}).PlaylistId(id).MaxResults(50).PageToken(page)
		resYt, _ := call.Do()

		if err != nil || resYt.Items[0] == nil {
			log.Println(err)
			SendSimpleMessageResponse(
				s,
				i,
				"Couldn't fetch playlist items. Check if it's public.",
				models.ColorError,
			)
			return
		}

		if resYt.NextPageToken == "" {
			cont = false
		} else {
			page = resYt.NextPageToken
		}

		for _, v := range resYt.Items {
			list = append(list, VideoInfo{
				ID:        v.ContentDetails.VideoId,
				Title:     v.Snippet.Title,
				Author:    v.Snippet.ChannelTitle,
				Duration:  "",
				Thumbnail: v.Snippet.Thumbnails.Default.Url,
			})
		}
	}

	if instance.Voice.Connection == nil { // if there's already a voice connection
		instance.Voice.startAudioSession(s, i, channelId) //start a new session
	}

	SendSimpleMessageResponse(
		s,
		i,
		"Adding playlist. It may take some time ...\n\n"+urlPlaylist,
		models.ColorDefault,
	)

	globalError := false

	for _, entry := range list {
		if skip > 0 {
			skip--
			continue
		}

		if err != nil {
			globalError = true
		}

		song := Song{
			url:       "https://www.youtube.com/watch?v=" + entry.ID,
			videoInfo: entry,
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
