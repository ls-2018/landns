package testutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPResponse is response type from HTTPServer.
type HTTPResponse struct {
	Method      string
	Path        string
	RequestBody string
	Status      int
	Body        string
}

// Assert is assertion test to HTTP response.
func (h HTTPResponse) Assert(t SimpleTB, status int, body string) {
	t.Helper()

	if h.Status != status {
		t.Errorf("%s %s: unexpected status code: expected %d but got %d", h.Method, h.Path, status, h.Status)
	}

	if h.Body != body {
		t.Errorf("%s %s: unexpected response body:\nexpected:\n%s\nbut got:\n%s", h.Method, h.Path, body, h.Body)
	}
}

// HTTPServer is tester for HTTP blackbox test.
type HTTPServer struct {
	URL *url.URL
}

// StartHTTPServer is make http.Server and start it.
func StartHTTPServer(ctx context.Context, t SimpleTB, handler http.Handler) HTTPServer {
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: FindEmptyPort()}

	u, err := url.Parse(fmt.Sprintf("http://%s", addr))
	if err != nil {
		t.Fatalf("failed to make URL: %s", err)
	}

	server := http.Server{
		Addr:    addr.String(),
		Handler: handler,
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve HTTP server: %s", err)
		}
	}()

	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := server.Shutdown(c); err != nil {
			t.Fatalf("failed to stop HTTP server: %s", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // Wait for start DNS server

	return HTTPServer{u}
}

// Do is do HTTP request to server.
func (h HTTPServer) Do(t SimpleTB, method, path, body string) (r HTTPResponse) {
	t.Helper()

	u, err := h.URL.Parse(path)
	if err != nil {
		t.Errorf("failed to %s %s: %s", method, path, err)
		return
	}

	req, err := http.NewRequest(method, u.String(), strings.NewReader(body))
	if err != nil {
		t.Errorf("failed to %s %s: %s", method, path, err)
		return
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Errorf("failed to %s %s: %s", method, path, err)
		return
	}
	defer resp.Body.Close()

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("failed to %s %s: %s", method, path, err)
		return
	}

	return HTTPResponse{
		Method:      method,
		Path:        path,
		RequestBody: body,
		Status:      resp.StatusCode,
		Body:        string(rbody),
	}
}
