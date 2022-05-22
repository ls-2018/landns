package httplog

import (
	"net/http"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/logtest"
)

func TestResponseWriter_passthrough(t *testing.T) {
	upstream := new(ResponseWriteCounter)
	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to make request: %s", err)
	}
	w := &responseWriter{
		Logger:   new(logtest.DummyLogger),
		Upstream: upstream,
		Request:  req,
	}

	assert := func(t *testing.T, name string, expect, got int) {
		t.Helper()
		if got != expect {
			t.Errorf("unexpected %s: expected %d but got %d", name, expect, got)
		}
	}

	assert(t, "Header call count", 0, upstream.HeaderCount)
	w.Header()
	assert(t, "Header call count", 1, upstream.HeaderCount)

	assert(t, "Write call count", 0, upstream.WriteCount)
	l, _ := w.Write([]byte("hello"))
	assert(t, "Write response", 5, l)
	assert(t, "Write call count", 1, upstream.WriteCount)

	assert(t, "WriteHeader call count", 0, upstream.WriteHeaderCount)
	w.WriteHeader(http.StatusOK)
	assert(t, "WriteHeader call count", 1, upstream.WriteHeaderCount)
}

func TestResponseWriter_logging(t *testing.T) {
	tests := []struct {
		Method   string
		Host     string
		Path     string
		User     string
		Password string
		UA       string
		Referer  string
		Status   int
		LogLevel logger.Level
	}{
		{"GET", "example.com", "/test", "", "", "landns-test-agent", "http://example.com/from", 200, logger.InfoLevel},
		{"POST", "www.example.com", "/path/to", "macrat", "secret", "dummy user agent", "http://example.com/path/to", 302, logger.InfoLevel},
		{"DELETE", "example.com", "/", "", "", "landns-test-agent", "", 403, logger.WarnLevel},
		{"GET", "example.com", "", "", "", "", "", 500, logger.ErrorLevel},
	}

	for _, tt := range tests {
		log := new(logtest.DummyLogger)
		req, err := http.NewRequest(tt.Method, "http://"+tt.Host+tt.Path, nil)
		if err != nil {
			t.Errorf("failed to make request: %s", err)
			continue
		}
		req.Header.Set("User-Agent", tt.UA)
		req.Header.Set("Referer", tt.Referer)
		if tt.User != "" {
			req.SetBasicAuth(tt.User, tt.Password)
		}

		w := &responseWriter{
			Logger:   log,
			Upstream: new(ResponseWriteCounter),
			Request:  req,
		}

		w.WriteHeader(tt.Status)

		err = log.TestAll([]logtest.Entry{
			{tt.LogLevel, tt.Method + " http://" + tt.Host + tt.Path, logger.Fields{
				"proto":      "HTTP/1.1",
				"method":     tt.Method,
				"host":       tt.Host,
				"user":       tt.User,
				"path":       tt.Path,
				"user_agent": tt.UA,
				"referer":    tt.Referer,
				"remote":     "",
				"status":     tt.Status,
			}},
		})
		if err != nil {
			t.Error(err)
		}
	}
}
