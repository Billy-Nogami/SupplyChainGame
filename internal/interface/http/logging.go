package http

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var requestSequence uint64

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(data)
	r.size += n

	return n, err
}

func WithRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := atomic.AddUint64(&requestSequence, 1)
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		log.Printf(
			"request_started request_id=%d method=%s path=%s remote_addr=%s user_agent=%q",
			requestID,
			r.Method,
			r.URL.RequestURI(),
			r.RemoteAddr,
			r.UserAgent(),
		)

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		log.Printf(
			"request_finished request_id=%d method=%s path=%s status=%d bytes=%d duration=%s",
			requestID,
			r.Method,
			r.URL.RequestURI(),
			status,
			recorder.size,
			time.Since(startedAt).Round(time.Millisecond),
		)
	})
}

func WithRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic_recovered method=%s path=%s err=%v", r.Method, r.URL.RequestURI(), recovered)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
