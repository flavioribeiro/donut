package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/flavioribeiro/donut/internal/entity"
)

type MediaHandler struct{}

func NewMediaHandler() *MediaHandler {
	return &MediaHandler{}
}

func (m *MediaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetCORS(w, r)
	if r.Method != http.MethodGet {
		ErrorToHTTP(w, entity.ErrHTTPGetOnly)
		return
	}

	offer := entity.ParamsOffer{}
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	log.Println("Connecting to SRT ", offer)
	_, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(300),
			astisrt.WithStreamid(offer.SRTStreamID),
		},

		// Callback when the connection is disconnected
		OnDisconnect: func(c *astisrt.Connection, err error) { log.Fatal("Disconnected from SRT") },

		Host: offer.SRTHost,
		Port: offer.SRTPort,
	})
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}
	log.Println("Connected to SRT")
}
