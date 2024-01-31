package controllers

import (
	"context"
	"net"

	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type WebRTCController struct {
	c   *entities.Config
	l   *zap.Logger
	api *webrtc.API
}

func NewWebRTCController(
	c *entities.Config,
	l *zap.Logger,
	api *webrtc.API,
) *WebRTCController {
	return &WebRTCController{
		c:   c,
		l:   l,
		api: api,
	}
}

func (c *WebRTCController) CreatePeerConnection(cancel context.CancelFunc) (*webrtc.PeerConnection, error) {
	c.l.Sugar().Infow("trying to set up web rtc conn")

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
		c.l.Sugar().Errorw("error while creating a new peer connection",
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
			c.l.Sugar().Infow("Canceling webrtc",
				"status", connectionState.String(),
			)
			cancel()
		}

		c.l.Sugar().Infow("OnICEConnectionStateChange",
			"status", connectionState.String(),
		)
	})

	return peerConnection, nil
}

func (c *WebRTCController) CreateTrack(peer *webrtc.PeerConnection, track entities.Track, id string, streamId string) (*webrtc.TrackLocalStaticSample, error) {
	codecCapability := mapper.FromTrackToRTPCodecCapability(track)
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

	c.l.Sugar().Infow("Gathering WebRTC Candidates")
	gatherComplete := webrtc.GatheringCompletePromise(peer)
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	} else if err = peer.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-gatherComplete
	c.l.Sugar().Infow("Gathering WebRTC Candidates Complete")

	return peer.LocalDescription(), nil
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
