package landns

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// HTTPError is error message of HTTP method.
type HTTPError struct {
	StatusCode int
	Message    string
}

// ServeHTTP is behave as http.Handler.
func (e HTTPError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(e.StatusCode)
	fmt.Fprintln(w, e.Error())
}

// Error is make string for response to client.
func (e HTTPError) Error() string {
	return fmt.Sprintf("; %d: %s", e.StatusCode, strings.ReplaceAll(e.Message, "\n", "\n;      "))
}

func parseRecordSet(req, remote string) (DynamicRecordSet, *HTTPError) {
	for _, x := range []struct {
		From string
		To   string
	}{
		{"$ADDR", remote},
		{"$TTL", "3600"},
		{"$$", "$"},
	} {
		req = strings.ReplaceAll(req, x.From, x.To)
	}

	rs, err := NewDynamicRecordSet(req)
	if err != nil {
		return nil, &HTTPError{http.StatusBadRequest, err.Error()}
	}
	return rs, nil
}

type httpHandler func(path string, body string, remote string) (string, *HTTPError)

func (hh httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		HTTPError{http.StatusBadRequest, "bad request"}.ServeHTTP(w, r)
		return
	}

	addr, _ := net.ResolveTCPAddr("tcp", r.RemoteAddr)

	resp, e := hh(r.URL.Path, string(body), addr.IP.String())
	if e != nil {
		e.ServeHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)

	resp = strings.TrimRight(resp, "\n")
	if len(resp) != 0 {
		fmt.Fprintln(w, resp)
	}
}

type httpHandlerSet map[string]http.Handler

func (hhs httpHandlerSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := hhs[r.Method]; ok {
		h.ServeHTTP(w, r)
		return
	}

	HTTPError{http.StatusMethodNotAllowed, "method not allowed"}.ServeHTTP(w, r)
}

// DynamicAPI is API request handler.
type DynamicAPI struct {
	Resolver DynamicResolver
}

func (d DynamicAPI) GetAllRecords(path, req, remote string) (string, *HTTPError) {
	records, err := d.Resolver.Records()
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return records.String(), nil
}

func (d DynamicAPI) GetRecordByID(path, req, remote string) (string, *HTTPError) {
	id, err := strconv.Atoi(path[len("/v1/id/"):])
	if err != nil {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	records, err := d.Resolver.GetRecord(id)
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	if len(records) == 0 {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	return records.String(), nil
}

func (d DynamicAPI) GetRecordsBySuffix(path, req, remote string) (string, *HTTPError) {
	if path[len(path)-1] == '/' {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	items := strings.Split(path[len("/v1/suffix/"):], "/")
	rev := make([]string, len(items))
	for i := range items {
		rev[i] = items[len(items)-1-i]
	}
	domain := Domain(strings.Join(rev, "."))

	if err := domain.Validate(); err != nil || domain.String()[0] == '.' {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	records, err := d.Resolver.SearchRecords(domain)
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return records.String(), nil
}

func (d DynamicAPI) GetRecordsByGlob(path, req, remote string) (string, *HTTPError) {
	glob := path[len("/v1/glob/"):]
	if strings.Contains(glob, "/") || len(glob) == 0 {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	if glob[len(glob)-1] != '.' {
		glob += "."
	}

	records, err := d.Resolver.GlobRecords(glob)
	if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return records.String(), nil
}

func (d DynamicAPI) setRecords(rs DynamicRecordSet) (string, *HTTPError) {
	if err := d.Resolver.SetRecords(rs); err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	add := 0
	del := 0
	for _, r := range rs {
		if r.Disabled {
			del++
		} else {
			add++
		}
	}

	return fmt.Sprintf("; 200: add:%d delete:%d", add, del), nil
}

func (d DynamicAPI) PostRecords(path, req, remote string) (string, *HTTPError) {
	rs, err := parseRecordSet(req, remote)
	if err != nil {
		return "", err
	}

	return d.setRecords(rs)
}

func (d DynamicAPI) DeleteRecords(path, req, remote string) (string, *HTTPError) {
	rs, err := parseRecordSet(req, remote)
	if err != nil {
		return "", err
	}

	for i := range rs {
		rs[i].Disabled = !rs[i].Disabled
	}

	return d.setRecords(rs)
}

func (d DynamicAPI) DeleteRecordByID(path, req, remote string) (string, *HTTPError) {
	id, err := strconv.Atoi(path[len("/v1/id/"):])
	if err != nil {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	}

	if err := d.Resolver.RemoveRecord(id); err == ErrNoSuchRecord {
		return "", &HTTPError{http.StatusNotFound, "not found"}
	} else if err != nil {
		return "", &HTTPError{http.StatusInternalServerError, "internal server error"}
	}

	return "; 200: ok", nil
}

func (d DynamicAPI) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/v1", httpHandlerSet{
		"GET":    httpHandler(d.GetAllRecords),
		"POST":   httpHandler(d.PostRecords),
		"DELETE": httpHandler(d.DeleteRecords),
	})
	mux.Handle("/v1/id/", httpHandlerSet{
		"GET":    httpHandler(d.GetRecordByID),
		"DELETE": httpHandler(d.DeleteRecordByID),
	})
	mux.Handle("/v1/suffix/", httpHandlerSet{"GET": httpHandler(d.GetRecordsBySuffix)})
	mux.Handle("/v1/glob/", httpHandlerSet{"GET": httpHandler(d.GetRecordsByGlob)})
	mux.Handle("/", HTTPError{http.StatusNotFound, "not found"})

	return mux
}
