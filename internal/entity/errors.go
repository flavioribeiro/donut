package entity

import "errors"

var ErrHTTPGetOnly = errors.New("you must use http GET verb")
var ErrHTTPPostOnly = errors.New("you must use http POST verb")
var ErrMissingParamsOffer = errors.New("ParamsOffer must not be nil")
var ErrMissingSRTHost = errors.New("SRTHost must not be nil")
var ErrMissingSRTPort = errors.New("SRTPort must be valid")
var ErrMissingSRTStreamID = errors.New("SRTStreamID must not be empty")
var ErrMissingWebRTCSetup = errors.New("WebRTCController.SetupPeerConnection must be called first")
