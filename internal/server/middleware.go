package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dgfront/bmax2/internal/maxauth"
)

// InitDataHeader carries the raw WebApp.initData string from the mini app.
const InitDataHeader = "X-Max-Init-Data"

type initDataKey struct{}

func (s *Server) requireInitData(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := s.validator.Validate(r.Header.Get(InitDataHeader))
		if err != nil {
			s.log.WarnContext(r.Context(), "initData rejected", "err", err, "ip", clientIP(r))
			writeError(w, http.StatusUnauthorized, "invalid initData")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), initDataKey{}, data)))
	})
}

func initDataFrom(ctx context.Context) maxauth.InitData {
	d, _ := ctx.Value(initDataKey{}).(maxauth.InitData)
	return d
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		level := slog.LevelDebug
		if rec.status >= 500 {
			level = slog.LevelError
		}
		s.log.Log(r.Context(), level, "request",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration", time.Since(start), "ip", clientIP(r))
	})
}

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.ErrorContext(r.Context(), "panic", "path", r.URL.Path, "panic", v)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the address set by the reverse proxy, if any.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
