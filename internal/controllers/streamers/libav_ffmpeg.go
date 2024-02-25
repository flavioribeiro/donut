package streamers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/asticode/go-astikit"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LibAVFFmpegStreamer struct {
	c *entities.Config
	l *zap.SugaredLogger

	lastAudioFrameDTS     float64
	currentAudioFrameSize float64
}

type LibAVFFmpegStreamerParams struct {
	fx.In
	C *entities.Config
	L *zap.SugaredLogger
}

type ResultLibAVFFmpegStreamer struct {
	fx.Out
	LibAVFFmpegStreamer DonutStreamer `group:"streamers"`
}

func NewLibAVFFmpegStreamer(p LibAVFFmpegStreamerParams) ResultLibAVFFmpegStreamer {
	return ResultLibAVFFmpegStreamer{
		LibAVFFmpegStreamer: &LibAVFFmpegStreamer{
			c: p.C,
			l: p.L,
		},
	}
}

func (c *LibAVFFmpegStreamer) Match(req *entities.RequestParams) bool {
	return req.SRTHost != ""
}

type streamContext struct {
	inputStream     *astiav.Stream
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	decFrame        *astiav.Frame
}

type params struct {
	inputFormatContext *astiav.FormatContext
	streams            map[int]*streamContext
}

func (c *LibAVFFmpegStreamer) Stream(donut *entities.DonutParameters) {
	c.l.Infow("streaming has started")

	closer := astikit.NewCloser()
	defer closer.Close()

	p := &params{
		streams: make(map[int]*streamContext),
	}

	if err := c.prepareInput(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	pkt := astiav.AllocPacket()
	closer.Add(pkt.Free)

	for {
		select {
		case <-donut.Ctx.Done():
			if errors.Is(donut.Ctx.Err(), context.Canceled) {
				c.l.Infow("streaming has stopped due cancellation")
				return
			}
			c.onError(donut.Ctx.Err(), donut)
			return
		default:

			if err := p.inputFormatContext.ReadFrame(pkt); err != nil {
				if errors.Is(err, astiav.ErrEof) {
					break
				}
				c.onError(err, donut)
			}

			s, ok := p.streams[pkt.StreamIndex()]
			if !ok {
				continue
			}
			pkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())

			audioDuration := c.defineAudioDuration(s, pkt)
			videoDuration := c.defineVideoDuration(s, pkt)

			if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
				if donut.OnVideoFrame != nil {
					if err := donut.OnVideoFrame(pkt.Data(), entities.MediaFrameContext{
						PTS:      int(pkt.Pts()),
						DTS:      int(pkt.Dts()),
						Duration: videoDuration,
					}); err != nil {
						c.onError(err, donut)
						return
					}
				}
			}

			if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeAudio {
				if donut.OnAudioFrame != nil {
					donut.OnAudioFrame(pkt.Data(), entities.MediaFrameContext{
						PTS:      int(pkt.Pts()),
						DTS:      int(pkt.Dts()),
						Duration: audioDuration,
					})
				}
			}
		}
	}
}

func (c *LibAVFFmpegStreamer) onError(err error, p *entities.DonutParameters) {
	if p.OnError != nil {
		p.OnError(err)
	}
}

func (c *LibAVFFmpegStreamer) prepareInput(p *params, closer *astikit.Closer, donut *entities.DonutParameters) error {
	// good for debugging
	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		c.l.Infof("ffmpeg log: %s (level: %d)", strings.TrimSpace(msg), l)
	})

	if p.inputFormatContext = astiav.AllocFormatContext(); p.inputFormatContext == nil {
		return errors.New("ffmpeg/libav: input format context is nil")
	}
	closer.Add(p.inputFormatContext.Free)

	inputFormat, err := c.defineInputFormat(donut.StreamFormat)
	if err != nil {
		return err
	}
	inputOptions := c.defineInputOptions(donut, closer)
	if err := p.inputFormatContext.OpenInput(donut.StreamURL, inputFormat, inputOptions); err != nil {
		return errors.New(fmt.Sprintf("ffmpeg/libav: opening input failed %s", err.Error()))
	}

	closer.Add(p.inputFormatContext.CloseInput)

	if err := p.inputFormatContext.FindStreamInfo(nil); err != nil {
		return errors.New(fmt.Sprintf("ffmpeg/libav: finding stream info failed %s", err.Error()))
	}

	for _, is := range p.inputFormatContext.Streams() {
		if is.CodecParameters().MediaType() != astiav.MediaTypeAudio &&
			is.CodecParameters().MediaType() != astiav.MediaTypeVideo {
			c.l.Infof("skipping media type %s", is.CodecParameters().MediaType().String())
			continue
		}

		s := &streamContext{inputStream: is}

		if s.decCodec = astiav.FindDecoder(is.CodecParameters().CodecID()); s.decCodec == nil {
			return errors.New("ffmpeg/libav: codec is nil")
		}

		if s.decCodecContext = astiav.AllocCodecContext(s.decCodec); s.decCodecContext == nil {
			return errors.New("ffmpeg/libav: codec context is nil")
		}
		closer.Add(s.decCodecContext.Free)

		if err := is.CodecParameters().ToCodecContext(s.decCodecContext); err != nil {
			return errors.New(fmt.Sprintf("ffmpeg/libav: updating codec context failed %s", err.Error()))
		}

		if is.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			s.decCodecContext.SetFramerate(p.inputFormatContext.GuessFrameRate(is, nil))
		}

		if err := s.decCodecContext.Open(s.decCodec, nil); err != nil {
			return errors.New(fmt.Sprintf("ffmpeg/libav: opening codec context failed %s", err.Error()))
		}

		s.decFrame = astiav.AllocFrame()
		closer.Add(s.decFrame.Free)

		p.streams[is.Index()] = s
	}
	return nil
}

func (c *LibAVFFmpegStreamer) defineInputFormat(streamFormat string) (*astiav.InputFormat, error) {
	if streamFormat != "" {
		inputFormat := astiav.FindInputFormat(streamFormat)
		if inputFormat == nil {
			return nil, errors.New(fmt.Sprintf("ffmpeg/libav: could not find %s input format", streamFormat))
		}
	}
	return nil, nil
}

func (c *LibAVFFmpegStreamer) defineInputOptions(p *entities.DonutParameters, closer *astikit.Closer) *astiav.Dictionary {
	if strings.Contains(strings.ToLower(p.StreamURL), "srt:") {
		d := &astiav.Dictionary{}
		closer.Add(d.Free)

		// ref https://ffmpeg.org/ffmpeg-all.html#srt
		// flags (the zeroed 3rd value) https://github.com/FFmpeg/FFmpeg/blob/n5.0/libavutil/dict.h#L67C9-L77
		d.Set("srt_streamid", p.StreamID, 0)
		d.Set("smoother", "live", 0)
		d.Set("transtype", "live", 0)
		return d
	}
	return nil
}

func (c *LibAVFFmpegStreamer) defineAudioDuration(s *streamContext, pkt *astiav.Packet) time.Duration {
	audioDuration := time.Duration(0)
	if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeAudio {
		// Audio
		//
		// dur = 0,023219954648526078
		// sample = 44100
		// frameSize = 1024 (or 960 for aac, but it could be variable for opus)
		// 1s = dur * (sample/frameSize)
		// ref https://developer.apple.com/documentation/coreaudiotypes/audiostreambasicdescription/1423257-mframesperpacket

		// TODO: handle wraparound
		c.currentAudioFrameSize = float64(pkt.Dts()) - c.lastAudioFrameDTS
		c.lastAudioFrameDTS = float64(pkt.Dts())
		sampleRate := float64(s.inputStream.CodecParameters().SampleRate())
		audioDuration = time.Duration((c.currentAudioFrameSize / sampleRate) * float64(time.Second))
	}
	return audioDuration
}

func (c *LibAVFFmpegStreamer) defineVideoDuration(s *streamContext, pkt *astiav.Packet) time.Duration {
	videoDuration := time.Duration(0)
	if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
		// Video
		//
		// dur = 0,033333
		// sample = 30
		// frameSize = 1
		// 1s = dur * (sample/frameSize)

		// we're assuming fixed video frame rate
		videoDuration = time.Duration((float64(1) / float64(s.inputStream.AvgFrameRate().Num())) * float64(time.Second))
	}
	return videoDuration
}
