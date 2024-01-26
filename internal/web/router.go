package handlers

import "net/http"

func NewServeMux(
	index *IndexHandler,
	signaling *SignalingHandler,
) *http.ServeMux {

	mux := http.NewServeMux()

	mux.Handle("/", index)
	mux.Handle("/doSignaling", signaling)

	return mux
}
