package httplog

import (
	"net/http"
)

type ResponseWriteCounter struct {
	HeaderCount      int
	WriteCount       int
	WriteHeaderCount int
}

func (w *ResponseWriteCounter) Header() http.Header {
	w.HeaderCount++
	return nil
}

func (w *ResponseWriteCounter) Write(b []byte) (int, error) {
	w.WriteCount++
	return len(b), nil
}

func (w *ResponseWriteCounter) WriteHeader(statusCode int) {
	w.WriteHeaderCount++
}
