package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/flavioribeiro/donut/internal/entities"
)

type MediaHandler struct{}

func NewMediaHandler() *MediaHandler {
	return &MediaHandler{}
}

func (m *MediaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SetError(w, entities.ErrHTTPGetOnly)
		return
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		SetError(w, err)
		return
	}

	log.Println("Connecting to SRT ", params)
	_, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(300),
			astisrt.WithStreamid(params.SRTStreamID),
		},

		// Callback when the connection is disconnected
		OnDisconnect: func(c *astisrt.Connection, err error) { log.Fatal("Disconnected from SRT") },

		Host: params.SRTHost,
		Port: params.SRTPort,
	})
	if err != nil {
		SetError(w, err)
		return
	}
	log.Println("Connected to SRT")
}
