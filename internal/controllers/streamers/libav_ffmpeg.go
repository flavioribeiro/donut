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
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LibAVFFmpegStreamer struct {
	c *entities.Config
	l *zap.SugaredLogger
	m *mapper.Mapper

	lastAudioFrameDTS     float64
	currentAudioFrameSize float64
}

type LibAVFFmpegStreamerParams struct {
	fx.In
	C *entities.Config
	L *zap.SugaredLogger
	M *mapper.Mapper
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
			m: p.M,
		},
	}
}

func (c *LibAVFFmpegStreamer) Match(req *entities.RequestParams) bool {
	return req.SRTHost != ""
}

type streamContext struct {
	// IN
	inputStream     *astiav.Stream
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	decFrame        *astiav.Frame

	// OUT
	encCodec        *astiav.Codec
	encCodecContext *astiav.CodecContext
	encPkt          *astiav.Packet
}

type libAVParams struct {
	inputFormatContext *astiav.FormatContext
	streams            map[int]*streamContext
}

func (c *LibAVFFmpegStreamer) Stream(donut *entities.DonutParameters) {
	c.l.Infow("streaming has started")

	closer := astikit.NewCloser()
	defer closer.Close()

	p := &libAVParams{
		streams: make(map[int]*streamContext),
	}

	// it's really useful for debugging
	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		c.l.Infof("ffmpeg %s: - %s", c.libAVLogToString(l), strings.TrimSpace(msg))
	})

	if err := c.prepareInput(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	// the audio codec opus expects 48000 (for webrtc), therefore filters are needed
	// so one can upscale 44100 to 48000 frames/samples through filters
	// https://ffmpeg.org/ffmpeg-filters.html#aformat
	// https://ffmpeg.org/ffmpeg-filters.html#aresample-1
	// https://github.com/FFmpeg/FFmpeg/blob/8b6219a99d80cabf87c50170c009fe93092e32bd/doc/examples/resample_audio.c#L133
	// https://github.com/FFmpeg/FFmpeg/blob/8b6219a99d80cabf87c50170c009fe93092e32bd/doc/examples/mux.c#L295
	// ffmpeg error: more samples than frame size

	if err := c.prepareOutput(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	inPkt := astiav.AllocPacket()
	closer.Add(inPkt.Free)

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

			if err := p.inputFormatContext.ReadFrame(inPkt); err != nil {
				if errors.Is(err, astiav.ErrEof) {
					break
				}
				c.onError(err, donut)
			}

			s, ok := p.streams[inPkt.StreamIndex()]
			if !ok {
				continue
			}
			// TODO: understand why it's necessary
			inPkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())

			isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
			isVideoBypass := donut.Recipe.Video.Action == entities.DonutBypass
			if isVideo && isVideoBypass {
				if donut.OnVideoFrame != nil {
					if err := donut.OnVideoFrame(inPkt.Data(), entities.MediaFrameContext{
						PTS:      int(inPkt.Pts()),
						DTS:      int(inPkt.Dts()),
						Duration: c.defineVideoDuration(s, inPkt),
					}); err != nil {
						c.onError(err, donut)
						return
					}
				}
				continue
			}

			isAudio := s.decCodecContext.MediaType() == astiav.MediaTypeAudio
			isAudioBypass := donut.Recipe.Audio.Action == entities.DonutBypass
			if isAudio && isAudioBypass {
				if donut.OnAudioFrame != nil {
					if err := donut.OnAudioFrame(inPkt.Data(), entities.MediaFrameContext{
						PTS:      int(inPkt.Pts()),
						DTS:      int(inPkt.Dts()),
						Duration: c.defineAudioDuration(s, inPkt),
					}); err != nil {
						c.onError(err, donut)
						return
					}
				}
				continue
			}

			// send the coded packet (compressed/encoded frame) to the decoder
			if err := s.decCodecContext.SendPacket(inPkt); err != nil {
				c.onError(err, donut)
				return
			}

			for {
				// receive the raw frame from the decoder
				if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
					if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
						break
					}
					c.onError(err, donut)
					return
				}
				// send the raw frame to the encoder
				if err := c.encodeFrame(s.decFrame, s, donut); err != nil {
					c.onError(err, donut)
					return
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

func (c *LibAVFFmpegStreamer) prepareInput(p *libAVParams, closer *astikit.Closer, donut *entities.DonutParameters) error {
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
		return fmt.Errorf("ffmpeg/libav: opening input failed %w", err)
	}
	closer.Add(p.inputFormatContext.CloseInput)

	if err := p.inputFormatContext.FindStreamInfo(nil); err != nil {
		return fmt.Errorf("ffmpeg/libav: finding stream info failed %w", err)
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
			return fmt.Errorf("ffmpeg/libav: updating codec context failed %w", err)
		}

		if is.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			s.decCodecContext.SetFramerate(p.inputFormatContext.GuessFrameRate(is, nil))
		}

		if err := s.decCodecContext.Open(s.decCodec, nil); err != nil {
			return fmt.Errorf("ffmpeg/libav: opening codec context failed %w", err)
		}

		s.decFrame = astiav.AllocFrame()
		closer.Add(s.decFrame.Free)

		p.streams[is.Index()] = s

		if donut.OnStream != nil {
			stream := c.m.FromLibAVStreamToEntityStream(is)
			donut.OnStream(&stream)
		}
	}
	return nil
}

func (c *LibAVFFmpegStreamer) prepareOutput(p *libAVParams, closer *astikit.Closer, donut *entities.DonutParameters) error {
	for _, is := range p.inputFormatContext.Streams() {
		s, ok := p.streams[is.Index()]
		if !ok {
			c.l.Infof("skipping stream index = %d", is.Index())
			continue
		}

		isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
		isVideoBypass := donut.Recipe.Video.Action == entities.DonutBypass
		if isVideo && isVideoBypass {
			c.l.Infof("skipping video transcoding for %+v", s.inputStream)
			continue
		}

		isAudio := s.decCodecContext.MediaType() == astiav.MediaTypeAudio
		isAudioBypass := donut.Recipe.Audio.Action == entities.DonutBypass
		if isAudio && isAudioBypass {
			c.l.Infof("skipping audio transcoding for %+v", s.inputStream)
			continue
		}

		var codecID astiav.CodecID
		if isAudio {
			audioCodecID, err := c.m.FromStreamCodecToLibAVCodecID(donut.Recipe.Audio.Codec)
			if err != nil {
				return err
			}
			codecID = audioCodecID
		}
		if isVideo {
			videoCodecID, err := c.m.FromStreamCodecToLibAVCodecID(donut.Recipe.Video.Codec)
			if err != nil {
				return err
			}
			codecID = videoCodecID
		}

		if s.encCodec = astiav.FindEncoder(codecID); s.encCodec == nil {
			// TODO: migrate error to entity
			return fmt.Errorf("cannot find a libav encoder for %+v", codecID)
		}

		if s.encCodecContext = astiav.AllocCodecContext(s.encCodec); s.encCodecContext == nil {
			// TODO: migrate error to entity
			return errors.New("ffmpeg/libav: codec context is nil")
		}
		closer.Add(s.encCodecContext.Free)

		if isAudio {
			if v := s.encCodec.ChannelLayouts(); len(v) > 0 {
				s.encCodecContext.SetChannelLayout(v[0])
			} else {
				s.encCodecContext.SetChannelLayout(s.decCodecContext.ChannelLayout())
			}
			s.encCodecContext.SetChannels(s.decCodecContext.Channels())
			s.encCodecContext.SetSampleRate(s.decCodecContext.SampleRate())
			if v := s.encCodec.SampleFormats(); len(v) > 0 {
				s.encCodecContext.SetSampleFormat(v[0])
			} else {
				s.encCodecContext.SetSampleFormat(s.decCodecContext.SampleFormat())
			}
			s.encCodecContext.SetTimeBase(s.decCodecContext.TimeBase())

			// supplying custom config
			if len(donut.Recipe.Audio.CodecContextOptions) > 0 {
				for _, opt := range donut.Recipe.Audio.CodecContextOptions {
					opt(s.encCodecContext)
				}
			}
		}

		if isVideo {
			if v := s.encCodec.PixelFormats(); len(v) > 0 {
				s.encCodecContext.SetPixelFormat(v[0])
			} else {
				s.encCodecContext.SetPixelFormat(s.decCodecContext.PixelFormat())
			}
			s.encCodecContext.SetSampleAspectRatio(s.decCodecContext.SampleAspectRatio())
			s.encCodecContext.SetTimeBase(s.decCodecContext.TimeBase())
			s.encCodecContext.SetHeight(s.decCodecContext.Height())
			s.encCodecContext.SetWidth(s.decCodecContext.Width())
			// s.encCodecContext.SetFramerate(p.inputFormatContext.GuessFrameRate(s.inputStream, nil))
			s.encCodecContext.SetFramerate(s.inputStream.AvgFrameRate())

			// supplying custom config
			if len(donut.Recipe.Audio.CodecContextOptions) > 0 {
				for _, opt := range donut.Recipe.Audio.CodecContextOptions {
					opt(s.encCodecContext)
				}
			}
		}

		if s.decCodecContext.Flags().Has(astiav.CodecContextFlagGlobalHeader) {
			s.encCodecContext.SetFlags(s.encCodecContext.Flags().Add(astiav.CodecContextFlagGlobalHeader))
		}

		if err := s.encCodecContext.Open(s.encCodec, nil); err != nil {
			return fmt.Errorf("opening encoder context failed: %w", err)
		}

		s.encPkt = astiav.AllocPacket()
		closer.Add(s.encPkt.Free)

		// // Update codec parameters
		// if err = s.outputStream.CodecParameters().FromCodecContext(s.encCodecContext); err != nil {
		// 	err = fmt.Errorf("main: updating codec parameters failed: %w", err)
		// 	return
		// }

		// // Update stream
		// s.outputStream.SetTimeBase(s.encCodecContext.TimeBase())
	}
	return nil
}

func (c *LibAVFFmpegStreamer) encodeFrame(f *astiav.Frame, s *streamContext, donut *entities.DonutParameters) (err error) {
	// Reset picture type
	f.SetPictureType(astiav.PictureTypeNone)

	s.encPkt.Unref()

	// Send frame
	if err = s.encCodecContext.SendFrame(f); err != nil {
		err = fmt.Errorf("main: sending frame failed: %w", err)
		return
	}

	// Loop
	for {
		// Receive packet
		if err = s.encCodecContext.ReceivePacket(s.encPkt); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				err = nil
				break
			}
			err = fmt.Errorf("main: receiving packet failed: %w", err)
			return
		}

		// Update pkt
		// 		s.encPkt.RescaleTs(s.encCodecContext.TimeBase(), s.outputStream.TimeBase())
		s.encPkt.RescaleTs(s.encCodecContext.TimeBase(), s.decCodecContext.TimeBase())

		isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
		if isVideo {
			if donut.OnVideoFrame != nil {
				if err := donut.OnVideoFrame(s.encPkt.Data(), entities.MediaFrameContext{
					PTS:      int(s.encPkt.Pts()),
					DTS:      int(s.encPkt.Dts()),
					Duration: c.defineVideoDuration(s, s.encPkt),
				}); err != nil {
					return err
				}
			}
		}

		isAudio := s.decCodecContext.MediaType() == astiav.MediaTypeAudio
		if isAudio {
			if donut.OnAudioFrame != nil {
				if err := donut.OnAudioFrame(s.encPkt.Data(), entities.MediaFrameContext{
					PTS:      int(s.encPkt.Pts()),
					DTS:      int(s.encPkt.Dts()),
					Duration: c.defineAudioDuration(s, s.encPkt),
				}); err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func (c *LibAVFFmpegStreamer) defineInputFormat(streamFormat string) (*astiav.InputFormat, error) {
	if streamFormat != "" {
		inputFormat := astiav.FindInputFormat(streamFormat)
		if inputFormat == nil {
			return nil, fmt.Errorf("ffmpeg/libav: could not find %s input format", streamFormat)
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

		// TODO: properly handle wraparound / roll over
		// or explore av frame_size https://ffmpeg.org/doxygen/trunk/structAVCodecContext.html#aec57f0d859a6df8b479cd93ca3a44a33
		// and libAV pts roll over
		if float64(pkt.Dts())-c.lastAudioFrameDTS > 0 {
			c.currentAudioFrameSize = float64(pkt.Dts()) - c.lastAudioFrameDTS
		}

		c.lastAudioFrameDTS = float64(pkt.Dts())
		sampleRate := float64(s.inputStream.CodecParameters().SampleRate())
		audioDuration = time.Duration((c.currentAudioFrameSize / sampleRate) * float64(time.Second))
		c.l.Infow("audio duration",
			"framesize", s.inputStream.CodecParameters().FrameSize(),
			"audioDuration", audioDuration,
		)
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
		c.l.Infow("video duration",
			"framesize", s.inputStream.CodecParameters().FrameSize(),
			"videoDuration", videoDuration,
		)
	}
	return videoDuration
}

// TODO: move this either to a mapper or make a PR for astiav
func (*LibAVFFmpegStreamer) libAVLogToString(l astiav.LogLevel) string {
	const _Ciconst_AV_LOG_DEBUG = 0x30
	const _Ciconst_AV_LOG_ERROR = 0x10
	const _Ciconst_AV_LOG_FATAL = 0x8
	const _Ciconst_AV_LOG_INFO = 0x20
	const _Ciconst_AV_LOG_PANIC = 0x0
	const _Ciconst_AV_LOG_QUIET = -0x8
	const _Ciconst_AV_LOG_VERBOSE = 0x28
	const _Ciconst_AV_LOG_WARNING = 0x18
	switch l {
	case _Ciconst_AV_LOG_WARNING:
		return "WARN"
	case _Ciconst_AV_LOG_VERBOSE:
		return "VERBOSE"
	case _Ciconst_AV_LOG_QUIET:
		return "QUIET"
	case _Ciconst_AV_LOG_PANIC:
		return "PANIC"
	case _Ciconst_AV_LOG_INFO:
		return "INFO"
	case _Ciconst_AV_LOG_FATAL:
		return "FATAL"
	case _Ciconst_AV_LOG_DEBUG:
		return "DEBUG"
	case _Ciconst_AV_LOG_ERROR:
		return "ERROR"
	default:
		return "UNKNOWN LEVEL"
	}
}
