package streamers

import "github.com/flavioribeiro/donut/internal/entities"

type DonutStreamer interface {
	Stream(sp *entities.StreamParameters)
	Match(req *entities.RequestParams) bool
}
