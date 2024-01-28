package controllers

import (
	"net"

	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type WebRTCController struct {
	c      *entities.Config
	l      *zap.Logger
	iceTcp net.Listener
	iceUdp net.PacketConn
	peer   *webrtc.PeerConnection
}

func NewWebRTCController(
	c *entities.Config,
	l *zap.Logger,
	iceTcp net.Listener,
	iceUdp net.PacketConn,
) *WebRTCController {
	return &WebRTCController{
		c:      c,
		l:      l,
		iceTcp: iceTcp,
		iceUdp: iceUdp,
	}
}

func (c *WebRTCController) SetupPeerConnection() error {
	if c.peer != nil {
		return nil
	}

	peerConnectionConfiguration := webrtc.Configuration{}
	if !c.c.EnableICEMux {
		peerConnectionConfiguration.ICEServers = []webrtc.ICEServer{
			{
				URLs: c.c.StunServers,
			},
		}
	}

	mediaEngine, err := NewWebRTCMediaEngine()
	if err != nil {
		return err
	}

	api := webrtc.NewAPI(
		webrtc.WithSettingEngine(NewWebRTCSettingsEngine(
			c.c,
			c.iceTcp,
			c.iceUdp,
		)),
		webrtc.WithMediaEngine(mediaEngine),
	)

	peerConnection, err := api.NewPeerConnection(peerConnectionConfiguration)
	if err != nil {
		return err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		c.l.Sugar().Infow("OnICEConnectionStateChange",
			"status", connectionState.String(),
		)
	})
	c.peer = peerConnection
	return nil
}

func (c *WebRTCController) CreateTrack(track entities.Track, id string, streamId string) (*webrtc.TrackLocalStaticSample, error) {
	codecCapability := mapper.FromTrackToRTPCodecCapability(track)
	webRTCtrack, err := webrtc.NewTrackLocalStaticSample(codecCapability, id, streamId)
	if err != nil {
		return nil, err
	}

	if _, err := c.peer.AddTrack(webRTCtrack); err != nil {
		return nil, err
	}
	return webRTCtrack, nil
}

func (c *WebRTCController) CreateDataChannel(channelID string) (*webrtc.DataChannel, error) {
	if c.peer == nil {
		// TODO: or call SetupPeerConnection?
		return nil, entities.ErrMissingWebRTCSetup
	}

	metadataSender, err := c.peer.CreateDataChannel(channelID, nil)
	if err != nil {
		return nil, err
	}
	return metadataSender, nil
}

func (c *WebRTCController) SetRemoteDescription(desc webrtc.SessionDescription) error {
	if c.peer == nil {
		// TODO: or call SetupPeerConnection?
		return entities.ErrMissingWebRTCSetup
	}

	err := c.peer.SetRemoteDescription(desc)
	if err != nil {
		return err
	}
	return nil
}

func (c *WebRTCController) GatheringWebRTC() (*webrtc.SessionDescription, error) {
	if c.peer == nil {
		// TODO: or call SetupPeerConnection?
		return nil, entities.ErrMissingWebRTCSetup
	}

	c.l.Sugar().Infow("Gathering WebRTC Candidates")
	gatherComplete := webrtc.GatheringCompletePromise(c.peer)
	answer, err := c.peer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	} else if err = c.peer.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-gatherComplete

	c.l.Sugar().Infow("Gathering WebRTC Candidates Complete")
	return c.peer.LocalDescription(), nil
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
