package httplog

import (
	"net/http"

	"github.com/macrat/landns/lib-landns/logger"
)

// responseWriter is wrapper of http.ResponseWriter for logging WriteHeader.
type responseWriter struct {
	Logger   logger.Logger
	Upstream http.ResponseWriter
	Request  *http.Request
	logged   bool
}

// Header is just call w.Upstream.Header.
func (w *responseWriter) Header() http.Header {
	return w.Upstream.Header()
}

// Write is just call w.Upstream.Write.
func (w *responseWriter) Write(b []byte) (int, error) {
	w.logging(http.StatusOK)
	return w.Upstream.Write(b)
}

func (w *responseWriter) logging(statusCode int) {
	if w.logged {
		return
	}

	user, _, _ := w.Request.BasicAuth()

	logFunc := w.Logger.Info
	switch {
	case 400 <= statusCode && statusCode <= 499:
		logFunc = w.Logger.Warn
	case 500 <= statusCode && statusCode <= 599:
		logFunc = w.Logger.Error
	}
	logFunc(w.Request.Method+" "+w.Request.URL.String(), logger.Fields{
		"proto":      w.Request.Proto,
		"method":     w.Request.Method,
		"host":       w.Request.Host,
		"user":       user,
		"path":       w.Request.URL.Path,
		"user_agent": w.Request.UserAgent(),
		"referer":    w.Request.Referer(),
		"remote":     w.Request.RemoteAddr,
		"status":     statusCode,
	})

	w.logged = true
}

// WriteHeader is logging response status code and call w.Upstream.WriteHeader.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.logging(statusCode)
	w.Upstream.WriteHeader(statusCode)
}
