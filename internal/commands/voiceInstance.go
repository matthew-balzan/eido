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
	QueueList  []Song //Copy of the channel, needed to show queue to the user
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
	i.QueueList = make([]Song, 0, models.MaxQueueLength)
	return i
}

func (v *VoiceInstance) PlaySingleSong(videoInfo *youtube.Video) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.Volume = 0.3

	client := youtube.Client{}

	formats := videoInfo.Formats.WithAudioChannels()
	streamYT, _, err := client.GetStream(videoInfo, &formats[0])
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Error getting the stream - ", err)
		return
	}

	encodingSession, err := dca.EncodeMem(streamYT, options)
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

	v.Encoder = nil
	v.Stream = nil

	defer encodingSession.Cleanup()
	v.Connection.Speaking(false)

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
		SendSimpleMessage(s, i, "Disconnected for inactivity", models.ColorDefault)
	}()
}

func (v *VoiceInstance) startAudioSession(s *discordgo.Session, i *discordgo.InteractionCreate, voiceChannel string) {
	v.Queue = make(chan Song, models.MaxQueueLength)

	var err error = nil
	var voiceConnection *discordgo.VoiceConnection = nil
	voiceConnection, err = s.ChannelVoiceJoin(i.GuildID, voiceChannel, false, true)

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
				models.ColorDefault,
				"Now playing:",
			)

			isConnectionReady := v.Connection.Ready

			for i := 0; !isConnectionReady && i < 6; i++ { // retry 6 times, which is equals to 30 seconds
				time.Sleep(5 * time.Second)
			}

			v.PlaySingleSong(song.videoInfo)

			if len(v.QueueList) > 0 { // in case a clear has happened
				v.QueueList = v.QueueList[1:] // dequeue
			}
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
	v.QueueList = append(v.QueueList, song)
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
	v.QueueList = make([]Song, 0, models.MaxQueueLength)
	v.skip()
}

func (v *VoiceInstance) getQueueList() (queue []Song) {
	return v.QueueList
}
