package streamers

import "github.com/flavioribeiro/donut/internal/entities"

type DonutStreamer interface {
	Stream(p *entities.DonutParameters)
	Match(req *entities.RequestParams) bool
}
