package main

import (
	"errors"
	"net/http"
	"strconv"
)

func assertSignalingCorrect(SRTHost, SRTPort, SRTStreamID string) (int, error) {
	switch {
	case SRTHost == "":
		return 0, errors.New("SRTHost must not be nil")
	case SRTPort == "":
		return 0, errors.New("SRTPort must not be empty")
	case SRTStreamID == "":
		return 0, errors.New("SRTStreamID must not be empty")
	}

	return strconv.Atoi(SRTPort)
}

func errorToHTTP(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}

func setCors(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		allowedHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token"
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
	}
}
