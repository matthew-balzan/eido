package commands

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
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

func (v *VoiceInstance) PlaySingleSong(url string) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.AudioFilter = "volume=0.1"
	options.BufferedFrames = 1024 * 1024 * 4

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", "-f", "best*[vcodec=none][acodec=opus]", "-o", "-", "--download-sections", "*from-url", url)
	defer cmd.Wait()

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err != nil {
		log.Println("ERR: internal/models/instance.go: Error calling os exec yt-dlp - ", err)
	}
	if err := cmd.Start(); err != nil {
		log.Println("ERR: internal/models/instance.go: Error starting the yt-dlp command - ", err)
	}

	buf := bufio.NewReaderSize(stdout, 8*1024*1024)

	encodingSession, err := dca.EncodeMem(buf, options)
	if err != nil {
		log.Println("ERR: internal/models/instance.go: Error encoding - ", err)
		return
	}
	defer encodingSession.Cleanup()

	v.Encoder = encodingSession

	done := make(chan error)

	v.Connection.Speaking(true)

	var stream = dca.NewStream(encodingSession, v.Connection, done)
	v.Stream = stream
	errDone := <-done
	cancel()

	v.Encoder = nil
	v.Stream = nil

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
			if v.Connection == nil {
				return
			}

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

			for i := 0; !v.Connection.Ready && i < 6; i++ { // retry 6 times, which is equals to 30 seconds
				time.Sleep(5 * time.Second)
			}

			v.PlaySingleSong(song.url)

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
