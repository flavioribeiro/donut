package probers

import "github.com/flavioribeiro/donut/internal/entities"

type DonutProber interface {
	StreamInfo(req *entities.RequestParams) (*entities.StreamInfo, error)
	Match(req *entities.RequestParams) bool
}
