package httplog_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/httplog"
	"github.com/macrat/landns/lib-landns/logger/logtest"
	"github.com/macrat/landns/lib-landns/testutil"
)

func TestHTTPLogger(t *testing.T) {
	log := new(logtest.DummyLogger)
	handler := httplog.HTTPLogger{Logger: log, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := testutil.StartHTTPServer(ctx, t, handler)

	r := srv.Do(t, "GET", "/somewhere", "")
	r.Assert(t, 200, "hello world")

	expect := logtest.Entry{
		Level:   logger.InfoLevel,
		Message: "GET /somewhere",
		Fields: logger.Fields{
			"proto":      "HTTP/1.1",
			"method":     "GET",
			"host":       srv.URL.Host,
			"user":       "",
			"path":       "/somewhere",
			"user_agent": "Go-http-client/1.1",
			"referer":    "",
			"remote":     "127.0.0.1:*",
			"status":     200,
		},
	}

	ok := len(*log) == 1

	if ok {
		l := (*log)[0]
		if l.Level != expect.Level || l.Message != expect.Message || len(l.Fields) != len(expect.Fields) {
			ok = false
		} else {
			for k, pattern := range expect.Fields {
				if m, err := filepath.Match(fmt.Sprint(pattern), fmt.Sprint(l.Fields[k])); err != nil {
					t.Errorf("failed to glob match fields: %s", err)
					ok = false
					break
				} else if !m {
					ok = false
					break
				}
			}
		}
	}

	if !ok {
		t.Error(log.TestAll([]logtest.Entry{expect}))
	}
}
