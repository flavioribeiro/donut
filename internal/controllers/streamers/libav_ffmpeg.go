package streamers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
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
	isRTMP := strings.Contains(strings.ToLower(req.StreamURL), "rtmp")
	isSRT := strings.Contains(strings.ToLower(req.StreamURL), "srt")

	return isRTMP || isSRT
}

type streamContext struct {
	// IN
	inputStream     *astiav.Stream
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	decFrame        *astiav.Frame

	// FILTER
	filterGraph       *astiav.FilterGraph
	buffersinkContext *astiav.FilterContext
	buffersrcContext  *astiav.FilterContext
	filterFrame       *astiav.Frame

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
	c.l.Infof("streaming has started for %#v", donut)

	closer := astikit.NewCloser()
	defer closer.Close()

	p := &libAVParams{
		streams: make(map[int]*streamContext),
	}

	// it's useful for debugging
	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(_ astiav.Classer, l astiav.LogLevel, fmt, msg string) {
		c.l.Infof("ffmpeg %s: - %s", c.libAVLogToString(l), strings.TrimSpace(msg))
	})
	// 138.1 internal/controllers/streamers/libav_ffmpeg.go:95:24:
	// cannot use func(l astiav.LogLevel, fmt, msg, parent string) {…}
	// (value of type func(l astiav.LogLevel, fmt string, msg string, parent string)) as astiav.LogCallback value in argument to astiav.SetLogCallback

	c.l.Infof("preparing input")
	if err := c.prepareInput(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	c.l.Infof("preparing output")
	if err := c.prepareOutput(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	c.l.Infof("preparing filters")
	if err := c.prepareFilters(p, closer, donut); err != nil {
		c.onError(err, donut)
		return
	}

	inPkt := astiav.AllocPacket()
	closer.Add(inPkt.Free)

	for {
		select {
		case <-donut.Ctx.Done():
			if errors.Is(donut.Ctx.Err(), context.Canceled) {
				c.l.Info("streaming has stopped due cancellation")
				return
			}
			c.onError(donut.Ctx.Err(), donut)
			return
		default:
			c.l.Infof("started reading frame")
			if err := p.inputFormatContext.ReadFrame(inPkt); err != nil {
				if errors.Is(err, astiav.ErrEof) {
					c.l.Info("streaming has ended")
					return
				}
				c.onError(err, donut)
			}

			s, ok := p.streams[inPkt.StreamIndex()]
			if !ok {
				c.l.Warnf("cannot find stream id=%d", inPkt.StreamIndex())
				continue
			}

			inPkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())

			isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
			isVideoBypass := donut.Recipe.Video.Action == entities.DonutBypass
			if isVideo && isVideoBypass {
				if donut.OnVideoFrame != nil {
					// The SRT(mpegts[h264]) bitstream format is Annex B  0x0, 0x0, 0x0, 0x1 [Start Code]
					//		[start code]--[NAL]--[start code]--[NAL] etc
					//
					// The RTMP(flv[h264]) bitstream format is AVCC (mp4) 0xY, 0xZ, 0xK, 0xW [Length]
					//		[SIZE (4 bytes)]--[NAL]--[SIZE (4 bytes)]--[NAL] etc
					//
					// ref: https://stackoverflow.com/questions/28421375/usage-of-start-code-for-h264-video/29103276#29103276
					//
					// To convert from AVCC to AnnexB:
					//
					// Remove length, insert start code, insert SPS for each I-frame, insert PPS for each frame, insert AU delimiter for each GOP.
					//
					// https://ffmpeg.org/doxygen/trunk/h264__mp4toannexb__bsf_8c.html#a773e34981d7642d499348d1ae72fd02e

					// av_bsf_send_packet(bsfContext, pkt)
					// av_bsf_receive_packet(bsfContext, pkt)

					// for {
					// 	c.l.Infof("start receiving packet")
					// 	if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
					// 		if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
					// 			break
					// 		}
					// 		c.onError(err, donut)
					// 		return
					// 	}
					// 	c.l.Infof("start filtering")
					// 	if err := c.filterAndEncode(s.decFrame, s, donut); err != nil {
					// 		c.onError(err, donut)
					// 		return
					// 	}
					// }

					bistreamFilter := astiav.FindBitStreamFilterByName("h264_mp4toannexb")
					if bistreamFilter == nil {
						c.l.Info("cannot find bit stream filter")
						return
					}
					bsfCtx, err := astiav.AllocBitStreamContext(bistreamFilter)
					if err != nil {
						c.l.Info("error while AllocBitStreamContext", err)
						return
					}
					if err := bsfCtx.Init(); err != nil {
						c.l.Info("error while init", err)
						return
					}
					if err := bsfCtx.SendPacket(inPkt); err != nil {
						c.l.Info("error while SendPacket", err)
						return
					}

					if bsfCtx.ReceivePacket(inPkt) != nil {
						c.l.Info("error while ReceivePacket", err)
						return
					}

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

			if isAudio {
				continue
			}

			c.l.Infof("start sending packet")
			// c.processPacket(inPkt, s, donut)
			if err := s.decCodecContext.SendPacket(inPkt); err != nil {
				c.onError(err, donut)
				return
			}

			for {
				c.l.Infof("start receiving packet")
				if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
					if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
						break
					}
					c.onError(err, donut)
					return
				}
				c.l.Infof("start filtering")
				if err := c.filterAndEncode(s.decFrame, s, donut); err != nil {
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

	inputFormat, err := c.defineInputFormat(donut.Recipe.Input.Format.String())
	if err != nil {
		return err
	}
	inputOptions := c.defineInputOptions(donut, closer)
	if err := p.inputFormatContext.OpenInput(donut.Recipe.Input.URL, inputFormat, inputOptions); err != nil {
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
			return errors.New("ffmpeg/libav: codec is missing")
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
			err := donut.OnStream(&stream)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func functionNameFor(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	components := strings.Split(fullName, ".")
	return components[len(components)-2]
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
			c.l.Infof("bypass video for %+v", s.inputStream)
			continue
		}

		isAudio := s.decCodecContext.MediaType() == astiav.MediaTypeAudio
		isAudioBypass := donut.Recipe.Audio.Action == entities.DonutBypass
		if isAudio && isAudioBypass {
			c.l.Infof("bypass audio for %+v", s.inputStream)
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
					c.l.Infof("overriding av codec context %s", functionNameFor(opt))
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
			s.encCodecContext.SetFramerate(s.inputStream.AvgFrameRate())

			// supplying custom config
			if len(donut.Recipe.Video.CodecContextOptions) > 0 {
				for _, opt := range donut.Recipe.Video.CodecContextOptions {
					c.l.Infof("overriding av codec context %s", functionNameFor(opt))
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
	}
	return nil
}

func (c *LibAVFFmpegStreamer) prepareFilters(p *libAVParams, closer *astikit.Closer, donut *entities.DonutParameters) error {
	for _, s := range p.streams {

		isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
		isVideoBypass := donut.Recipe.Video.Action == entities.DonutBypass
		if isVideo && isVideoBypass {
			c.l.Infof("bypass video for %+v", s.inputStream)
			continue
		}

		isAudio := s.decCodecContext.MediaType() == astiav.MediaTypeAudio
		isAudioBypass := donut.Recipe.Audio.Action == entities.DonutBypass
		if isAudio && isAudioBypass {
			c.l.Infof("bypass audio for %+v", s.inputStream)
			continue
		}

		var args astiav.FilterArgs
		var buffersrc, buffersink *astiav.Filter
		var content string
		var err error

		if s.filterGraph = astiav.AllocFilterGraph(); s.filterGraph == nil {
			return errors.New("main: graph is nil")
		}
		closer.Add(s.filterGraph.Free)

		outputs := astiav.AllocFilterInOut()
		if outputs == nil {
			return errors.New("main: outputs is nil")
		}
		closer.Add(outputs.Free)

		inputs := astiav.AllocFilterInOut()
		if inputs == nil {
			return errors.New("main: inputs is nil")
		}
		closer.Add(inputs.Free)

		if s.decCodecContext.MediaType() == astiav.MediaTypeAudio {
			args = astiav.FilterArgs{
				"channel_layout": s.decCodecContext.ChannelLayout().String(),
				"sample_fmt":     s.decCodecContext.SampleFormat().Name(),
				"sample_rate":    strconv.Itoa(s.decCodecContext.SampleRate()),
				"time_base":      s.decCodecContext.TimeBase().String(),
			}
			buffersrc = astiav.FindFilterByName("abuffer")
			buffersink = astiav.FindFilterByName("abuffersink")
			content = fmt.Sprintf(
				"aresample=%s", // necessary for opus
				strconv.Itoa(s.encCodecContext.SampleRate()),
			)
		}

		if s.decCodecContext.MediaType() == astiav.MediaTypeVideo {
			args = astiav.FilterArgs{
				"pix_fmt":      strconv.Itoa(int(s.decCodecContext.PixelFormat())),
				"pixel_aspect": s.decCodecContext.SampleAspectRatio().String(),
				"time_base":    s.decCodecContext.TimeBase().String(),
				"video_size":   strconv.Itoa(s.decCodecContext.Width()) + "x" + strconv.Itoa(s.decCodecContext.Height()),
			}
			buffersrc = astiav.FindFilterByName("buffer")
			buffersink = astiav.FindFilterByName("buffersink")
			content = fmt.Sprintf("format=pix_fmts=%s", s.encCodecContext.PixelFormat().Name())
		}

		if buffersrc == nil {
			return errors.New("main: buffersrc is nil")
		}
		if buffersink == nil {
			return errors.New("main: buffersink is nil")
		}

		if s.buffersrcContext, err = s.filterGraph.NewFilterContext(buffersrc, "in", args); err != nil {
			return fmt.Errorf("main: creating buffersrc context failed: %w", err)
		}
		if s.buffersinkContext, err = s.filterGraph.NewFilterContext(buffersink, "out", nil); err != nil {
			return fmt.Errorf("main: creating buffersink context failed: %w", err)
		}

		outputs.SetName("in")
		outputs.SetFilterContext(s.buffersrcContext)
		outputs.SetPadIdx(0)
		outputs.SetNext(nil)

		inputs.SetName("out")
		inputs.SetFilterContext(s.buffersinkContext)
		inputs.SetPadIdx(0)
		inputs.SetNext(nil)

		if err = s.filterGraph.Parse(content, inputs, outputs); err != nil {
			return fmt.Errorf("main: parsing filter failed: %w", err)
		}

		if err = s.filterGraph.Configure(); err != nil {
			return fmt.Errorf("main: configuring filter failed: %w", err)
		}

		s.filterFrame = astiav.AllocFrame()
		closer.Add(s.filterFrame.Free)

		s.encPkt = astiav.AllocPacket()
		closer.Add(s.encPkt.Free)
	}
	return nil
}

func (c *LibAVFFmpegStreamer) processPacket(pkt *astiav.Packet, s *streamContext, donut *entities.DonutParameters) {
	if err := s.decCodecContext.SendPacket(pkt); err != nil {
		c.onError(err, donut)
		return
	}

	for {
		c.l.Infof("start receiving packet")
		if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				break
			}
			c.onError(err, donut)
			return
		}
		c.l.Infof("start filtering")
		if err := c.filterAndEncode(s.decFrame, s, donut); err != nil {
			c.onError(err, donut)
			return
		}
	}
}

func (c *LibAVFFmpegStreamer) filterAndEncode(f *astiav.Frame, s *streamContext, donut *entities.DonutParameters) (err error) {
	if err = s.buffersrcContext.BuffersrcAddFrame(f, astiav.NewBuffersrcFlags(astiav.BuffersrcFlagKeepRef)); err != nil {
		return fmt.Errorf("adding frame failed: %w", err)
	}
	for {
		s.filterFrame.Unref()

		if err = s.buffersinkContext.BuffersinkGetFrame(s.filterFrame, astiav.NewBuffersinkFlags()); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				err = nil
				break
			}
			return fmt.Errorf("getting frame failed: %w", err)
		}
		// TODO: should we avoid setting the picture type for audio?
		s.filterFrame.SetPictureType(astiav.PictureTypeNone)
		c.l.Infof("start encoding")
		if err = c.encodeFrame(s.filterFrame, s, donut); err != nil {
			err = fmt.Errorf("main: encoding and writing frame failed: %w", err)
			return
		}
	}
	return nil
}

func (c *LibAVFFmpegStreamer) encodeFrame(f *astiav.Frame, s *streamContext, donut *entities.DonutParameters) (err error) {
	s.encPkt.Unref()

	// when converting from aac to opus using filters, the np samples are bigger than the frame size
	// to fix the error "more samples than frame size"
	f.SetNbSamples(s.encCodecContext.FrameSize())

	if err = s.encCodecContext.SendFrame(f); err != nil {
		return fmt.Errorf("sending frame failed: %w", err)
	}

	for {
		c.l.Infof("start receiving packet")
		if err = s.encCodecContext.ReceivePacket(s.encPkt); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				err = nil
				break
			}
			return fmt.Errorf("receiving packet failed: %w", err)
		}

		// TODO: check if we need to swap
		// pkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())
		s.encPkt.RescaleTs(s.inputStream.TimeBase(), s.encCodecContext.TimeBase())

		isVideo := s.decCodecContext.MediaType() == astiav.MediaTypeVideo
		if isVideo {
			if donut.OnVideoFrame != nil {
				c.l.Infof("sending transcoded video")
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
				c.l.Infof("sending transcoded audio")
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
	var inputFormat *astiav.InputFormat
	if streamFormat != "" {
		inputFormat = astiav.FindInputFormat(streamFormat)
		if inputFormat == nil {
			return nil, fmt.Errorf("ffmpeg/libav: could not find %s input format", streamFormat)
		}
	}
	return inputFormat, nil
}

func (c *LibAVFFmpegStreamer) defineInputOptions(p *entities.DonutParameters, closer *astikit.Closer) *astiav.Dictionary {
	var dic *astiav.Dictionary
	if len(p.Recipe.Input.Options) > 0 {
		dic = &astiav.Dictionary{}
		closer.Add(dic.Free)

		for k, v := range p.Recipe.Input.Options {
			dic.Set(k.String(), v, 0)
		}
	}
	return dic
}

func (c *LibAVFFmpegStreamer) defineAudioDuration(s *streamContext, pkt *astiav.Packet) time.Duration {
	audioDuration := time.Duration(0)
	if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeAudio {

		// Audio
		//
		// dur = 12.416666ms
		// sample = 48000
		// frameSize = 596 (it can be variable for opus)
		// 1s = dur * (sample/frameSize)
		// ref https://developer.apple.com/documentation/coreaudiotypes/audiostreambasicdescription/1423257-mframesperpacket

		// TODO: properly handle wraparound / roll over
		// or explore av frame_size https://ffmpeg.org/doxygen/trunk/structAVCodecContext.html#aec57f0d859a6df8b479cd93ca3a44a33
		// and libAV pts roll over
		if float64(pkt.Dts())-c.lastAudioFrameDTS > 0 {
			c.currentAudioFrameSize = float64(pkt.Dts()) - c.lastAudioFrameDTS
		}

		c.lastAudioFrameDTS = float64(pkt.Dts())
		sampleRate := float64(s.encCodecContext.SampleRate())
		audioDuration = time.Duration((c.currentAudioFrameSize / sampleRate) * float64(time.Second))
	}
	return audioDuration
}

func (c *LibAVFFmpegStreamer) defineVideoDuration(s *streamContext, _ *astiav.Packet) time.Duration {
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
