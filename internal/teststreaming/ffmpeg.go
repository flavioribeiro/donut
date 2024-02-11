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
	ExpectedStreams() []entities.Stream
	Output() FFmpegOutput
}
type FFmpegOutput struct {
	Host string
	Port int
}

type testFFmpeg struct {
	arguments       string
	expectedStreams []entities.Stream
	cmdExec         *exec.Cmd
	output          FFmpegOutput
}

func (t *testFFmpeg) Start() error {
	t.cmdExec = exec.Command("ffmpeg", prepareFFmpegParameters(t.arguments)...)
	// For debugging:
	// t.cmdExec.Stdout = os.Stdout
	// t.cmdExec.Stderr = os.Stderr

	go func() {
		if err := t.cmdExec.Run(); err != nil {
			if strings.Contains(err.Error(), "signal: killed") {
				return
			}
			log.Fatalln("XXXXXXXXXXXX error while running ffmpeg XXXXXXXXXXXX", err.Error())
			return
		}
	}()
	// TODO: check the output to determine whether the ffmpeg is ready to accept connections
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

func (t *testFFmpeg) ExpectedStreams() []entities.Stream {
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
