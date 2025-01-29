package service

import (
	"net/http"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code

		if code != http.StatusNotFound {
			w.ResponseWriter.WriteHeader(code)
			w.wroteHeader = true
		}
	}
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if w.statusCode == http.StatusNotFound {
		return len(b), nil
	}

	return w.ResponseWriter.Write(b)
}

func wrapNotFound(handler, notFoundHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrapper := responseWriterWrapper{ResponseWriter: w}
		handler.ServeHTTP(&wrapper, r)

		if wrapper.statusCode == http.StatusNotFound {
			notFoundHandler.ServeHTTP(w, r)
		}
	}
}
