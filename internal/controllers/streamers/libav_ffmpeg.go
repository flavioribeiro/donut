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
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LibAVFFmpegStreamer struct {
	c *entities.Config
	l *zap.SugaredLogger

	middlewares []entities.StreamMiddleware
}

type LibAVFFmpegStreamerParams struct {
	fx.In
	C *entities.Config
	L *zap.SugaredLogger

	Middlewares []entities.StreamMiddleware `group:"middlewares"`
}

type ResultLibAVFFmpegStreamer struct {
	fx.Out
	LibAVFFmpegStreamer DonutStreamer `group:"streamers"`
}

func NewLibAVFFmpegStreamer(p LibAVFFmpegStreamerParams) ResultLibAVFFmpegStreamer {
	return ResultLibAVFFmpegStreamer{
		LibAVFFmpegStreamer: &LibAVFFmpegStreamer{
			c:           p.C,
			l:           p.L,
			middlewares: p.Middlewares,
		},
	}
}

func (c *LibAVFFmpegStreamer) Match(req *entities.RequestParams) bool {
	if req.SRTHost != "" {
		return true
	}
	return false
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

func (c *LibAVFFmpegStreamer) Stream(sp *entities.StreamParameters) {
	c.l.Infow("streaming has started")

	closer := astikit.NewCloser()
	defer closer.Close()
	defer sp.WebRTCConn.Close()
	defer sp.Cancel()

	p := &params{
		streams: make(map[int]*streamContext),
	}

	if err := c.prepareInput(p, closer, sp); err != nil {
		c.l.Errorf("ffmpeg/libav: failed at prepareInput %s", err.Error())
		return
	}

	pkt := astiav.AllocPacket()
	closer.Add(pkt.Free)

	for {
		select {
		case <-sp.Ctx.Done():
			if errors.Is(sp.Ctx.Err(), context.Canceled) {
				c.l.Infow("streaming has stopped due cancellation")
				return
			}
			c.l.Errorw("streaming has stopped due errors",
				"error", sp.Ctx.Err(),
			)
			return
		default:

			if err := p.inputFormatContext.ReadFrame(pkt); err != nil {
				if errors.Is(err, astiav.ErrEof) {
					break
				}
				c.l.Fatalf("ffmpeg/libav: reading frame failed %s", err.Error())
			}

			s, ok := p.streams[pkt.StreamIndex()]
			if !ok {
				continue
			}
			pkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())

			if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
				if err := sp.VideoTrack.WriteSample(media.Sample{Data: pkt.Data(), Duration: time.Second / 30}); err != nil {
					c.l.Errorw("ffmpeg/libav: failed to write video to web rtc",
						"error", err,
					)
					return
				}
			}

			// if err := s.decCodecContext.SendPacket(pkt); err != nil {
			// 	c.l.Fatalf("ffmpeg/libav: sending packet failed %s", err.Error())
			// }

			// for {
			// 	if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
			// 		if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
			// 			break
			// 		}
			// 		c.l.Fatalf("ffmpeg/libav: receiving frame failed %s", err.Error())
			// 	}
			// }

		}
	}
}

func (c *LibAVFFmpegStreamer) prepareInput(p *params, closer *astikit.Closer, sp *entities.StreamParameters) error {
	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		c.l.Infof("ffmpeg log: %s (level: %d)", strings.TrimSpace(msg), l)
	})

	if p.inputFormatContext = astiav.AllocFormatContext(); p.inputFormatContext == nil {
		return errors.New("ffmpeg/libav: input format context is nil")
	}
	closer.Add(p.inputFormatContext.Free)

	// TODO: add an UI element for sub-type (format) when input is srt:// (defaulting to mpeg-ts)
	// We're assuming that SRT is carrying mpegts.
	userProvidedInputFormat := "mpegts"

	inputFormat := astiav.FindInputFormat(userProvidedInputFormat)
	if inputFormat == nil {
		return errors.New(fmt.Sprintf("ffmpeg/libav: could not find %s", userProvidedInputFormat))
	}

	d := &astiav.Dictionary{}
	// ref https://ffmpeg.org/ffmpeg-all.html#srt
	// flags (the zeroed 3rd value) https://github.com/FFmpeg/FFmpeg/blob/n5.0/libavutil/dict.h#L67C9-L77
	d.Set("srt_streamid", sp.RequestParams.SRTStreamID, 0)
	d.Set("smoother", "live", 0)
	d.Set("transtype", "live", 0)

	inputURL := fmt.Sprintf("srt://%s:%d", sp.RequestParams.SRTHost, sp.RequestParams.SRTPort)

	if err := p.inputFormatContext.OpenInput(inputURL, inputFormat, d); err != nil {
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
