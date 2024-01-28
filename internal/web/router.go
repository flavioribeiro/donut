package web

import (
	"net/http"

	"github.com/flavioribeiro/donut/internal/web/handlers"
)

func NewServeMux(
	index *handlers.IndexHandler,
	signaling *handlers.SignalingHandler,
) *http.ServeMux {

	mux := http.NewServeMux()

	mux.Handle("/", index)
	mux.Handle("/doSignaling", setCors(signaling))

	return mux
}

func setCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			allowedHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token"
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		}
		next.ServeHTTP(w, r)
	})
}
