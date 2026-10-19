// Package server exposes the mini app, its public API and the admin API.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/dgfront/bmax2/internal/catalog"
	"github.com/dgfront/bmax2/internal/maxauth"
	"github.com/dgfront/bmax2/internal/store"
)

type Options struct {
	Store      *store.Store
	Validator  *maxauth.Validator
	AdminToken string
	Web        fs.FS
	Logger     *slog.Logger
}

type Server struct {
	store      *store.Store
	validator  *maxauth.Validator
	adminToken string
	log        *slog.Logger
	snapshot   atomic.Pointer[catalog.Snapshot]
	static     *staticFiles
	handler    http.Handler
}

func New(ctx context.Context, opt Options) (*Server, error) {
	static, err := newStaticFiles(opt.Web)
	if err != nil {
		return nil, err
	}
	s := &Server{
		store:      opt.Store,
		validator:  opt.Validator,
		adminToken: opt.AdminToken,
		log:        opt.Logger,
		static:     static,
	}
	if err := s.reload(ctx); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /media/{name}", s.handleMedia)
	mux.Handle("GET /api/v1/catalog", s.requireInitData(http.HandlerFunc(s.handleCatalog)))
	mux.Handle("POST /api/v1/session", s.requireInitData(http.HandlerFunc(s.handleSession)))
	if s.adminToken != "" {
		s.routeAdmin(mux)
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.Handle("/", s.static)

	s.handler = s.recoverPanics(s.logRequests(securityHeaders(mux)))
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

// reload rebuilds the cached public catalog from the database.
func (s *Server) reload(ctx context.Context) error {
	doc, err := s.store.LoadDocument(ctx)
	if err != nil {
		return err
	}
	versions, err := s.store.MediaVersions(ctx)
	if err != nil {
		return err
	}
	snap, err := catalog.BuildSnapshot(doc, versions)
	if err != nil {
		return err
	}
	s.snapshot.Store(snap)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeStoreError maps store errors to HTTP responses.
func (s *Server) writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		s.log.ErrorContext(r.Context(), "store error", "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
