package probers

import "github.com/flavioribeiro/donut/internal/entities"

type DonutProber interface {
	StreamInfo(req entities.DonutAppetizer) (*entities.StreamInfo, error)
	Match(req *entities.RequestParams) bool
}
