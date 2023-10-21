package commands

import (
	"io"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"github.com/matthew-balzan/dca"
	"github.com/matthew-balzan/eido/internal/models"
)

type ServerInstance struct {
	ServerId string
	Voice    *VoiceInstance
}

type VoiceInstance struct {
	ChannelId  string
	Connection *discordgo.VoiceConnection
	Encoder    *dca.EncodeSession
	Stream     *dca.StreamingSession
	IsPlaying  bool
	Queue      chan Song
	Timer      *time.Timer
}

type Song struct {
	videoInfo *youtube.Video
	url       string
}

func CreateServerInstance(id string) (i *ServerInstance) {
	i = new(ServerInstance)
	i.ServerId = id
	i.Voice = CreateVoiceInstance()
	return i
}

func CreateVoiceInstance() (i *VoiceInstance) {
	i = new(VoiceInstance)
	i.ChannelId = ""
	i.Connection = nil
	i.Encoder = nil
	i.IsPlaying = false
	i.Timer = nil
	i.Queue = nil
	return i
}

func (v *VoiceInstance) PlaySingleSong(videoInfo *youtube.Video) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.Volume = 50

	client := youtube.Client{}

	formats := videoInfo.Formats.WithAudioChannels()
	streamUrl, err := client.GetStreamURL(videoInfo, &formats[0])
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Error getting the stream - ", err)
		return
	}

	encodingSession, err := dca.EncodeFile(streamUrl, options)
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Error encoding - ", err)
		return
	}

	v.Encoder = encodingSession

	done := make(chan error)

	v.Connection.Speaking(true)

	var stream = dca.NewStream(encodingSession, v.Connection, done)
	v.Stream = stream
	errDone := <-done

	v.Connection.Speaking(false)

	v.Encoder = nil
	v.Stream = nil

	defer encodingSession.Cleanup()

	if errDone != nil && errDone != io.EOF {
		log.Println("ERR: internal/models/instance.go: Error while playing - ", errDone)
		return
	}
}

func (v *VoiceInstance) StopTimer() {
	if v.Timer != nil {
		v.Timer.Stop()
	}
}

func (v *VoiceInstance) StartTimer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	v.Timer = time.NewTimer(time.Duration(models.TimeoutSecondsDisconnect) * time.Second)

	go func() {
		<-v.Timer.C // signal to disconnect

		log.Println("Bot disconnected for inactivity")
		v.disconnect()
		SendSimpleMessage(s, i, "Disconnected for inactivity")
	}()
}

func (v *VoiceInstance) startAudioSession(s *discordgo.Session, i *discordgo.InteractionCreate, voiceChannel string) {
	v.Queue = make(chan Song, models.MaxQueueLength)

	var err error = nil
	var voiceConnection *discordgo.VoiceConnection = nil
	voiceConnection, err = s.ChannelVoiceJoin(i.GuildID, voiceChannel, false, false)
	if err != nil {
		log.Println("ERR: internal/commands/audio.go: Error joining voice channel - ", err)
		return
	}

	v.ChannelId = voiceChannel
	v.Connection = voiceConnection

	go func() {
		v.StartTimer(s, i) // in case the first song will not be added because of an error
		for song := range v.Queue {
			v.StopTimer()

			v.IsPlaying = true

			SendComplexMessage(
				s,
				i,
				song.videoInfo.Title,
				song.url,
				song.videoInfo.Thumbnails[1].URL,
				"Duration: "+song.videoInfo.Duration.String(),
				models.ColorRed,
				"Now playing:",
			)

			v.PlaySingleSong(song.videoInfo)

			v.IsPlaying = false

			v.StartTimer(s, i)
		}

	}()
}

func (v *VoiceInstance) addToQueue(song Song) (res bool) {
	if v.Queue == nil {
		log.Println("ERR: internal/models/instance.go: Queue not initialized (somehow)")
		return false
	}

	if len(v.Queue) >= models.MaxQueueLength {
		return false
	}

	v.Queue <- song
	return true
}

func (v *VoiceInstance) skip() {
	if v.Encoder != nil {
		v.Encoder.Cleanup()
	}
	v.setPause(false)
}

func (v *VoiceInstance) setPause(pause bool) {
	if v.Stream != nil {
		v.Stream.SetPaused(pause)
	}
}

func (v *VoiceInstance) disconnect() {
	if v.Connection != nil {
		v.Connection.Disconnect()
	}
	v.Connection = nil
	v.ChannelId = ""
	v.Stream = nil
	v.Timer = nil
	if v.Queue != nil {
		close(v.Queue)
	}
}

func (v *VoiceInstance) clearQueue() {
	for len(v.Queue) > 0 {
		<-v.Queue
	}
	v.skip()
}
