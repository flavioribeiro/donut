package web

import (
	"net/http"

	"github.com/flavioribeiro/donut/internal/web/handlers"
	"go.uber.org/zap"
)

type ErrorHTTPHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

func NewServeMux(
	index *handlers.IndexHandler,
	signaling *handlers.SignalingHandler,
	l *zap.SugaredLogger,
) *http.ServeMux {

	mux := http.NewServeMux()

	mux.Handle("/", index)

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/demo/", setHTTPNoCaching(http.StripPrefix("/demo/", fs)))

	mux.Handle("/doSignaling", setCors(errorHandler(l, signaling)))

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

func errorHandler(l *zap.SugaredLogger, next ErrorHTTPHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := next.ServeHTTP(w, r)
		if err != nil {
			l.Errorw("error on handler",
				"err", err,
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func setHTTPNoCaching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, max-age=0, must-revalidate, proxy-revalidate")
		next.ServeHTTP(w, r)
	})
}
