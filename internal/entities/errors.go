package entities

import (
	"errors"
	"fmt"
)

var ErrHTTPGetOnly = errors.New("you must use http GET verb")
var ErrHTTPPostOnly = errors.New("you must use http POST verb")
var ErrMissingParamsOffer = errors.New("ParamsOffer must not be nil")

var ErrMissingStreamURL = errors.New("stream URL must not be nil")
var ErrMissingStreamID = errors.New("stream ID must not be nil")
var ErrUnsupportedStreamURL = errors.New("unsupported stream")

var ErrMissingSRTHost = errors.New("SRTHost must not be nil")
var ErrMissingSRTPort = errors.New("SRTPort must be valid")
var ErrMissingSRTStreamID = errors.New("SRTStreamID must not be empty")

var ErrMissingWebRTCSetup = errors.New("WebRTCController.SetupPeerConnection must be called first")
var ErrMissingRemoteOffer = errors.New("nil offer, in order to connect one must pass a valid offer")
var ErrMissingRequestParams = errors.New("RequestParams must not be nil")

var ErrMissingProcess = errors.New("there is no process running")
var ErrMissingProber = errors.New("there is no prober")
var ErrMissingStreamer = errors.New("there is no streamer")
var ErrMissingCompatibleStreams = errors.New("there is no compatible streams")

// FFmpeg/LibAV
var ErrFFMpegLibAV = errors.New("ffmpeg/libav error")
var ErrFFmpegLibAVNotFound = fmt.Errorf("%w input not found", ErrFFMpegLibAV)
var ErrFFmpegLibAVFormatContextIsNil = fmt.Errorf("%w format context is nil", ErrFFMpegLibAV)
var ErrFFmpegLibAVFormatContextOpenInputFailed = fmt.Errorf("%w format context open input has failed", ErrFFMpegLibAV)
var ErrFFmpegLibAVFindStreamInfo = fmt.Errorf("%w could not find stream info", ErrFFMpegLibAV)
