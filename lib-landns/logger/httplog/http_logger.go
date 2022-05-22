package httplog

import (
	"net/http"

	"github.com/macrat/landns/lib-landns/logger"
)

// HTTPLogger is logging middleware for http.Handler.
type HTTPLogger struct {
	Logger  logger.Logger
	Handler http.Handler
}

func (h HTTPLogger) getLogger() logger.Logger {
	if h.Logger == nil {
		return logger.GetLogger()
	}
	return h.Logger
}

// ServeHTTP is call h.Handler(w, r) and logging request/response.
func (h HTTPLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w2 := &responseWriter{Logger: h.getLogger(), Upstream: w, Request: r}
	h.Handler.ServeHTTP(w2, r)
}
