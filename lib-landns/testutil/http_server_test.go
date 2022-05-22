package testutil_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns/testutil"
)

func TestHTTPServer(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello world")
	})

	ctx, cancel := context.WithCancel(context.Background())
	srv := testutil.StartHTTPServer(ctx, t, handler)

	srv.Do(t, "GET", "/", "").Assert(t, 200, "hello world\n")

	cancel()
	time.Sleep(10 * time.Millisecond)

	tb := new(testutil.DummyTB)
	srv.Do(tb, "GET", "/", "")
	tb.AssertErrors(t, "failed to GET /: Get \"http://127.0.0.1:*/\": dial tcp 127.0.0.1:*: connect: connection refused")
	tb.AssertFatals(t)
}

func TestHTTPServer_Assert(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	handler.HandleFunc("/200", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello world")
	})
	handler.HandleFunc("/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "invalid request")
	})
	handler.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := testutil.StartHTTPServer(ctx, t, handler)

	tests := []struct {
		Assert       func(tb testutil.SimpleTB)
		ExpectErrors []string
		ExpectFatals []string
	}{
		{func(tb testutil.SimpleTB) {
			srv.Do(tb, "GET", "/200", "").Assert(tb, http.StatusOK, "hello world\n")
		}, nil, nil},
		{func(tb testutil.SimpleTB) {
			srv.Do(tb, "GET", "/400", "").Assert(tb, http.StatusBadRequest, "invalid request\n")
		}, nil, nil},
		{func(tb testutil.SimpleTB) {
			srv.Do(tb, "GET", "/500", "").Assert(tb, http.StatusInternalServerError, "")
		}, nil, nil},
		{func(tb testutil.SimpleTB) {
			srv.Do(tb, "GET", "/400", "").Assert(tb, http.StatusOK, "invalid request\n")
		}, []string{"GET /400: unexpected status code: expected 200 but got 400"}, nil},
		{func(tb testutil.SimpleTB) {
			srv.Do(tb, "GET", "/200", "").Assert(tb, http.StatusOK, "hello world!!")
		}, []string{strings.Join([]string{
			"GET /200: unexpected response body:",
			"expected:",
			"hello world!!",
			"but got:",
			"hello world",
			"",
		}, "\n")}, nil},
	}

	for _, tt := range tests {
		if len(tt.ExpectErrors) == 0 && len(tt.ExpectFatals) == 0 {
			tt.Assert(t)
		} else {
			tb := new(testutil.DummyTB)
			tt.Assert(tb)
			tb.AssertErrors(t, tt.ExpectErrors...)
			tb.AssertFatals(t, tt.ExpectFatals...)
		}
	}
}
