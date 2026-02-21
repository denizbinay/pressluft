package devserver

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

const placeholderMessage = "Wave 1 complete - features will be added incrementally"

type Server struct {
	httpServer *http.Server
	logger     *log.Logger
	addr       string
}

func New(addr string, logger *log.Logger) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(placeholderMessage + "\n"))
	})

	wrapped := requestLogger(logger, mux)

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: wrapped,
		},
		logger: logger,
		addr:   addr,
	}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Start() error {
	s.logger.Printf("event=startup addr=%s", s.addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func requestLogger(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now().UTC()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		elapsed := time.Since(started).Milliseconds()
		logger.Printf(
			"event=request ts=%s method=%s path=%s status=%d duration_ms=%d",
			started.Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			rec.statusCode,
			elapsed,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
