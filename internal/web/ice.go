package handlers

import (
	"net"

	"github.com/flavioribeiro/donut/internal/entity"
	"github.com/pion/webrtc/v3"
)

func NewTCPICEServer(c *entity.Config) (*net.TCPListener, error) {
	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: c.TCPICEPort,
	})
	if err != nil {
		return nil, err
	}
	return tcpListener, nil
}

func NewUDPICEServer(c *entity.Config) (*net.UDPConn, error) {

	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: c.UDPICEPort,
	})
	if err != nil {
		return nil, err
	}

	return udpListener, nil
}

func NewWebRTCSettingsEngine(c *entity.Config, tcpListener *net.TCPListener, udpListener *net.UDPConn) *webrtc.SettingEngine {
	settingEngine := &webrtc.SettingEngine{}

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
