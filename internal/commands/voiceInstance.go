package commands

import (
	"io"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
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
	Queue      []Song
	Timer      *time.Timer
}

type Song struct {
	Id    string
	Title string
	Url   string
	User  string
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
	i.Queue = []Song{}
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

	dca.NewStream(encodingSession, v.Connection, done)
	errDone := <-done

	v.Connection.Speaking(false)

	defer encodingSession.Cleanup()
	v.Encoder = nil

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
		<-v.Timer.C
		log.Println("Bot disconnected for inactivity")
		if v.Connection != nil {
			v.Connection.Disconnect()
		}

		v.Connection = nil
		v.ChannelId = ""
		v.Timer = nil

		SendSimpleMessage(s, i, "Disconnected for inactivity")
	}()
}
