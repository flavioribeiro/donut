package probers

import "github.com/flavioribeiro/donut/internal/entities"

type Prober interface {
	StreamInfo(req *entities.RequestParams) (map[entities.Codec]entities.Stream, error)
}
