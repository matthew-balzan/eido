package models

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
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
		fmt.Println("ERR: internal/models/instance.go: Error gettin the stream - ", err)
		return
	}

	encodingSession, err := dca.EncodeFile(streamUrl, options)
	if err != nil {
		fmt.Println("ERR: internal/models/instance.go: Error encoding - ", err)
		return
	}

	done := make(chan error)

	v.Connection.Speaking(true)

	dca.NewStream(encodingSession, v.Connection, done)

	errDone := <-done

	v.Connection.Speaking(false)

	defer encodingSession.Cleanup()

	if errDone != nil && errDone != io.EOF {
		fmt.Println("ERR: internal/models/instance.go: Error while playing - ", err)
		return
	}
}
