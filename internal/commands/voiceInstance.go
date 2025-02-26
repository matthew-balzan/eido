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
	"github.com/matthew-balzan/dca"
	"github.com/matthew-balzan/eido/internal/models"
	"github.com/matthew-balzan/eido/internal/utils"
)

type ServerInstance struct {
	ServerId string
	Voice    *VoiceInstance
}

type VoiceInstance struct {
	ctx    context.Context
	cancel context.CancelFunc

	ChannelId  string
	Connection *discordgo.VoiceConnection
	Encoder    *dca.EncodeSession
	Stream     *dca.StreamingSession
	IsPlaying  bool
	Queue      *utils.RingQueue[Song]
	Timer      *time.Timer
}

type VideoInfo struct {
	ID        string
	Title     string
	Author    string
	Duration  string
	Thumbnail string
}

type Song struct {
	videoInfo VideoInfo
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
	i.Queue = utils.NewRingQueue[Song](models.MaxQueueLength)
	return i
}

func (v *VoiceInstance) playSingleSong(url string) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.AudioFilter = "volume=0.1"
	options.BufferedFrames = 1024 * 1024 * 4
	options.Threads = 2

	ctx, cancel := context.WithCancel(v.ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", "-f", "best*[vcodec=none][acodec=opus]", "-o", "-", "--download-sections", "*from-url", "--no-playlist", url)
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

func (v *VoiceInstance) stopTimer() {
	if v.Timer != nil {
		v.Timer.Stop()
	}
}

func (v *VoiceInstance) startTimer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if v.Timer != nil {
		v.Timer.Stop()
		v.Timer = nil
	}
	v.Timer = time.NewTimer(time.Duration(models.TimeoutSecondsDisconnect) * time.Second)

	go func() {
		select {
		case <-v.Timer.C:
			log.Println("Bot disconnected for inactivity")
			v.disconnect()
			SendSimpleMessage(s, i, "Disconnected for inactivity", models.ColorDefault)
		case <-v.ctx.Done():
			// Context is done, time to go.
			return
		}
	}()
}

func (v *VoiceInstance) StartAudioSession(s *discordgo.Session, i *discordgo.InteractionCreate, voiceChannel string) {
	v.ctx, v.cancel = context.WithCancel(context.Background())
	v.Queue.Clear()

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
		v.startTimer(s, i) // in case the first song will not be added because of an error
		for {
			song, err := v.Queue.Accept(v.ctx)
			if err != nil {
				return
			}

			v.stopTimer()

			if v.Connection == nil {
				return
			}

			v.IsPlaying = true

			SendComplexMessage(
				s,
				i,
				song.videoInfo.Title,
				song.url,
				song.videoInfo.Thumbnail,
				song.videoInfo.Duration,
				models.ColorDefault,
				"Now playing:",
			)

			for i := 0; !v.Connection.Ready && i < 6; i++ { // retry 6 times, which is equals to 30 seconds
				time.Sleep(5 * time.Second)
			}

			v.playSingleSong(song.url)

			v.IsPlaying = false
			v.startTimer(s, i)
		}

	}()
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
	if v.cancel != nil {
		v.cancel()
	}
	v.ctx = nil
	v.cancel = nil

	if v.Connection != nil {
		v.Connection.Disconnect()
	}
	v.Connection = nil

	v.ChannelId = ""
	v.Stream = nil
	if v.Timer != nil {
		v.Timer.Stop()
	}
	v.Timer = nil

	if v.Queue != nil {
		v.Queue.Clear()
	}
}

func (v *VoiceInstance) clearQueue() {
	v.Queue.Clear()
	v.skip()
}
