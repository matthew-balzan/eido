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

func (v *VoiceInstance) PlaySingleSong(url string) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	client := youtube.Client{}

	fmt.Println("0")

	videoInfo, err := client.GetVideo(url)
	if err != nil {
		fmt.Println("ERR: internal/models/instance.go: Couldn't fetch video info - ", err)
		return
	}

	formats := videoInfo.Formats.WithAudioChannels()
	stream, _, err := client.GetStream(videoInfo, &formats[0])
	if err != nil {
		fmt.Println("ERR: internal/models/instance.go: Error gettin the stream - ", err)
		return
	}

	encodingSession, err := dca.EncodeMem(stream, options)
	if err != nil {
		fmt.Println("ERR: internal/models/instance.go: Error encoding - ", err)
		return
	}
	defer encodingSession.Cleanup()

	done := make(chan error)
	dca.NewStream(encodingSession, v.Connection, done)

	errDone := <-done
	if errDone != nil && errDone != io.EOF {
		fmt.Println("ERR: internal/models/instance.go: Error while playing - ", err)
		return
	}
}
