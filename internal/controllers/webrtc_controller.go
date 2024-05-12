package controllers

import (
	"context"
	"encoding/json"
	"net"

	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/zap"
)

type WebRTCController struct {
	c   *entities.Config
	l   *zap.SugaredLogger
	api *webrtc.API
	m   *mapper.Mapper
}

func NewWebRTCController(
	c *entities.Config,
	l *zap.SugaredLogger,
	api *webrtc.API,
	m *mapper.Mapper,
) *WebRTCController {
	return &WebRTCController{
		c:   c,
		l:   l,
		api: api,
		m:   m,
	}
}

func (c *WebRTCController) Setup(cancel context.CancelFunc, donutRecipe *entities.DonutRecipe, params entities.RequestParams) (*entities.WebRTCSetupResponse, error) {
	response := &entities.WebRTCSetupResponse{}
	peer, err := c.CreatePeerConnection(cancel)
	if err != nil {
		return nil, err
	}
	response.Connection = peer

	var videoTrack *webrtc.TrackLocalStaticSample
	videoTrack, err = c.CreateTrack(peer, donutRecipe.Video.Codec, string(entities.VideoType), params.StreamID)
	if err != nil {
		return nil, err
	}
	response.Video = videoTrack

	var audioTrack *webrtc.TrackLocalStaticSample
	audioTrack, err = c.CreateTrack(peer, donutRecipe.Audio.Codec, string(entities.AudioType), params.StreamID)
	if err != nil {
		return nil, err
	}
	response.Audio = audioTrack

	metadataSender, err := c.CreateDataChannel(peer, entities.MetadataChannelID)
	if err != nil {
		return nil, err
	}
	response.Data = metadataSender

	if err = c.SetRemoteDescription(peer, params.Offer); err != nil {
		return nil, err
	}

	localDescription, err := c.GatheringWebRTC(peer)
	if err != nil {
		return nil, err
	}
	response.LocalSDP = localDescription

	return response, nil
}

func (c *WebRTCController) CreatePeerConnection(cancel context.CancelFunc) (*webrtc.PeerConnection, error) {
	c.l.Infow("trying to set up web rtc conn")

	peerConnectionConfiguration := webrtc.Configuration{}
	if !c.c.EnableICEMux {
		peerConnectionConfiguration.ICEServers = []webrtc.ICEServer{
			{
				URLs: c.c.StunServers,
			},
		}
	}

	peerConnection, err := c.api.NewPeerConnection(peerConnectionConfiguration)
	if err != nil {
		c.l.Errorw("error while creating a new peer connection",
			"error", err,
		)
		return nil, err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		finished := connectionState == webrtc.ICEConnectionStateClosed ||
			connectionState == webrtc.ICEConnectionStateDisconnected ||
			connectionState == webrtc.ICEConnectionStateCompleted ||
			connectionState == webrtc.ICEConnectionStateFailed

		if finished {
			c.l.Infow("Canceling webrtc",
				"status", connectionState.String(),
			)
			cancel()
		}

		c.l.Infow("OnICEConnectionStateChange",
			"status", connectionState.String(),
		)
	})

	return peerConnection, nil
}

func (c *WebRTCController) CreateTrack(peer *webrtc.PeerConnection, codec entities.Codec, id string, streamId string) (*webrtc.TrackLocalStaticSample, error) {
	codecCapability := c.m.FromTrackToRTPCodecCapability(codec)
	webRTCtrack, err := webrtc.NewTrackLocalStaticSample(codecCapability, id, streamId)
	if err != nil {
		return nil, err
	}

	if _, err := peer.AddTrack(webRTCtrack); err != nil {
		return nil, err
	}
	return webRTCtrack, nil
}

func (c *WebRTCController) CreateDataChannel(peer *webrtc.PeerConnection, channelID string) (*webrtc.DataChannel, error) {
	metadataSender, err := peer.CreateDataChannel(channelID, nil)
	if err != nil {
		return nil, err
	}
	return metadataSender, nil
}

func (c *WebRTCController) SetRemoteDescription(peer *webrtc.PeerConnection, desc webrtc.SessionDescription) error {
	err := peer.SetRemoteDescription(desc)
	if err != nil {
		return err
	}
	return nil
}

func (c *WebRTCController) GatheringWebRTC(peer *webrtc.PeerConnection) (*webrtc.SessionDescription, error) {
	c.l.Infow("Gathering WebRTC Candidates")
	gatherComplete := webrtc.GatheringCompletePromise(peer)
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	} else if err = peer.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-gatherComplete
	c.l.Infow("Gathering WebRTC Candidates Complete")

	return peer.LocalDescription(), nil
}

func (c *WebRTCController) SendMediaSample(mediaTrack *webrtc.TrackLocalStaticSample, data []byte, mediaCtx entities.MediaFrameContext) error {
	if err := mediaTrack.WriteSample(media.Sample{Data: data, Duration: mediaCtx.Duration}); err != nil {
		return err
	}
	return nil
}

func (c *WebRTCController) SendMetadata(metaTrack *webrtc.DataChannel, st *entities.Stream) error {
	msg := c.m.FromStreamToEntityMessage(*st)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = metaTrack.SendText(string(msgBytes))
	if err != nil {
		return err
	}
	return nil
}

func NewWebRTCSettingsEngine(c *entities.Config, tcpListener net.Listener, udpListener net.PacketConn) webrtc.SettingEngine {
	settingEngine := webrtc.SettingEngine{}

	settingEngine.SetNAT1To1IPs(c.ICEExternalIPsDNAT, webrtc.ICECandidateTypeHost)
	settingEngine.SetICETCPMux(webrtc.NewICETCPMux(nil, tcpListener, c.ICEReadBufferSize))
	settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(nil, udpListener))

	return settingEngine
}

func NewWebRTCMediaEngine() (*webrtc.MediaEngine, error) {
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	return mediaEngine, nil
}

func NewWebRTCAPI(mediaEngine *webrtc.MediaEngine, settingEngine webrtc.SettingEngine) *webrtc.API {
	return webrtc.NewAPI(
		webrtc.WithSettingEngine(settingEngine),
		webrtc.WithMediaEngine(mediaEngine),
	)
}

func NewTCPICEServer(c *entities.Config) (net.Listener, error) {
	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: c.TCPICEPort,
	})
	if err != nil {
		return nil, err
	}
	return tcpListener, nil
}

func NewUDPICEServer(c *entities.Config) (net.PacketConn, error) {
	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: c.UDPICEPort,
	})
	if err != nil {
		return nil, err
	}
	return udpListener, nil
}
