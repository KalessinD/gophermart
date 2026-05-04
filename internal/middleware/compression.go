package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type (
	GzipWriter struct {
		http.ResponseWriter
		writer *gzip.Writer
	}
)

func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to read gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz // подменяем тело запроса
		}

		renderWriter := w
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gw := NewGzipWriter(w)
			defer gw.Close()
			renderWriter = gw
		}

		next.ServeHTTP(renderWriter, r)
	})
}

func NewGzipWriter(w http.ResponseWriter) *GzipWriter {
	return &GzipWriter{ResponseWriter: w, writer: nil}
}

func (g *GzipWriter) Write(b []byte) (int, error) {
	if g.writer == nil {
		g.writer = gzip.NewWriter(g.ResponseWriter)
	}

	return g.writer.Write(b)
}

func (g *GzipWriter) WriteHeader(statusCode int) {
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *GzipWriter) Close() error {
	if g.writer != nil {
		return g.writer.Close()
	}
	return nil
}
