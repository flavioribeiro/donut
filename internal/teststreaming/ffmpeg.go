package teststreaming

import (
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/flavioribeiro/donut/internal/entities"
)

const (
	ffmpeg_startup = 5 * time.Second
)

type FFmpeg interface {
	Start() error
	Stop() error
	ExpectedStreams() map[entities.Codec]entities.Stream
	Output() FFmpegOutput
}
type FFmpegOutput struct {
	Host string
	Port int
}

var FFMPEG_LIVE_SRT_MPEG_TS_H264_AAC = testFFmpeg{
	arguments: `
	-hide_banner -loglevel verbose
    	-re -f lavfi -i testsrc2=size=1280x720:rate=30,format=yuv420p
    	-f lavfi -i sine=frequency=1000:sample_rate=44100
    	-c:v libx264 -preset veryfast -tune zerolatency -profile:v baseline
    	-b:v 1000k -bufsize 2000k -x264opts keyint=30:min-keyint=30:scenecut=-1
    	-f mpegts srt://0.0.0.0:45678?mode=listener&smoother=live&transtype=live
	`,
	expectedStreams: map[entities.Codec]entities.Stream{
		entities.H264: entities.Stream{Codec: entities.H264, Type: entities.VideoType},
		entities.AAC:  entities.Stream{Codec: entities.AAC, Type: entities.AudioType},
	},
	output: FFmpegOutput{Host: "127.0.0.1", Port: 45678},
}

type testFFmpeg struct {
	arguments       string
	expectedStreams map[entities.Codec]entities.Stream
	cmdExec         *exec.Cmd
	output          FFmpegOutput
}

func (t *testFFmpeg) Start() error {
	t.cmdExec = exec.Command("ffmpeg", prepareFFmpegParameters(t.arguments)...)
	// Useful for debugging
	// t.cmdExec.Stdout = os.Stdout
	// t.cmdExec.Stderr = os.Stderr

	go func() {
		if err := t.cmdExec.Run(); err != nil {
			if strings.Contains(err.Error(), "signal: killed") {
				return
			}
			log.Fatalln("XXXXXXXXXXXX Error running ffmpeg XXXXXXXXXXXX", err.Error())
			return
		}
	}()
	time.Sleep(ffmpeg_startup)
	return nil
}

func (t *testFFmpeg) Stop() error {
	if t == nil || t.cmdExec == nil {
		return entities.ErrMissingProcess
	}

	if err := t.cmdExec.Process.Kill(); err != nil {
		return err
	}
	return nil
}

func (t *testFFmpeg) ExpectedStreams() map[entities.Codec]entities.Stream {
	return t.expectedStreams
}

func (t *testFFmpeg) Output() FFmpegOutput {
	return t.output
}

func prepareFFmpegParameters(cmd string) []string {
	result := []string{}

	for _, item := range strings.Split(cmd, " ") {
		item = strings.ReplaceAll(item, "\\", "")
		item = strings.ReplaceAll(item, "\n", "")
		item = strings.ReplaceAll(item, "\t", "")
		item = strings.ReplaceAll(item, " ", "")
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}
