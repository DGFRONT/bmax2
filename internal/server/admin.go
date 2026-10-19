package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dgfront/bmax2/internal/catalog"
	"github.com/dgfront/bmax2/internal/store"
)

const (
	maxJSONBody  = 4 << 20
	maxMediaBody = 2 << 20
)

func (s *Server) routeAdmin(mux *http.ServeMux) {
	handle := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, s.requireAdmin(h))
	}
	handle("GET /api/admin/stats", s.adminStats)
	handle("GET /api/admin/catalog", s.adminGetCatalog)
	handle("PUT /api/admin/catalog", s.adminPutCatalog)
	handle("PUT /api/admin/categories", s.adminOrderCategories)
	handle("PUT /api/admin/categories/{id}", s.adminPutCategory)
	handle("DELETE /api/admin/categories/{id}", s.adminDeleteCategory)
	handle("PUT /api/admin/organizations/{id}", s.adminPutOrganization)
	handle("DELETE /api/admin/organizations/{id}", s.adminDeleteOrganization)
	handle("PUT /api/admin/featured", s.adminPutFeatured)
	handle("PUT /api/admin/media/{name}", s.adminPutMedia)
	handle("DELETE /api/admin/media/{name}", s.adminDeleteMedia)
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	want := []byte("Bearer " + s.adminToken)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			s.log.WarnContext(r.Context(), "admin auth failed", "ip", clientIP(r), "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// decodeJSON reads a size-limited JSON body that must contain one value with
// only known fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "invalid JSON: trailing data")
		return false
	}
	return true
}

// mutate runs a store change, refreshes the public catalog and replies.
func (s *Server) mutate(w http.ResponseWriter, r *http.Request, status int, change func() error) {
	if err := change(); err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	if err := s.reload(r.Context()); err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	s.log.InfoContext(r.Context(), "admin change", "method", r.Method, "path", r.URL.Path)
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	writeJSON(w, status, map[string]string{"status": "ok"})
}

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats(r.Context())
	if err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) adminGetCatalog(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.LoadDocument(r.Context())
	if err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) adminPutCatalog(w http.ResponseWriter, r *http.Request) {
	var doc catalog.Document
	if !decodeJSON(w, r, &doc) {
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.ReplaceDocument(r.Context(), doc) })
}

func (s *Server) adminOrderCategories(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !decodeJSON(w, r, &ids) {
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.OrderCategories(r.Context(), ids) })
}

func (s *Server) adminPutCategory(w http.ResponseWriter, r *http.Request) {
	var c catalog.Category
	if !decodeJSON(w, r, &c) {
		return
	}
	if err := pathID(r, &c.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.PutCategory(r.Context(), c) })
}

func (s *Server) adminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	s.mutate(w, r, http.StatusNoContent, func() error { return s.store.DeleteCategory(r.Context(), r.PathValue("id")) })
}

func (s *Server) adminPutOrganization(w http.ResponseWriter, r *http.Request) {
	var o catalog.Organization
	if !decodeJSON(w, r, &o) {
		return
	}
	if err := pathID(r, &o.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.PutOrganization(r.Context(), o) })
}

func (s *Server) adminDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	s.mutate(w, r, http.StatusNoContent, func() error { return s.store.DeleteOrganization(r.Context(), r.PathValue("id")) })
}

func (s *Server) adminPutFeatured(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if !decodeJSON(w, r, &ids) {
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.SetFeatured(r.Context(), ids) })
}

func (s *Server) adminPutMedia(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !catalog.ValidMediaName(name) {
		writeError(w, http.StatusBadRequest, "name must be lowercase latin with .jpg, .png or .webp")
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMediaBody))
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "file is larger than 2 MB")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	want := store.MediaContentType(name)
	if got := http.DetectContentType(data); got != want {
		writeError(w, http.StatusBadRequest, "file content is "+got+", expected "+want)
		return
	}
	s.mutate(w, r, http.StatusOK, func() error { return s.store.PutMedia(r.Context(), name, want, data) })
}

func (s *Server) adminDeleteMedia(w http.ResponseWriter, r *http.Request) {
	s.mutate(w, r, http.StatusNoContent, func() error { return s.store.DeleteMedia(r.Context(), r.PathValue("name")) })
}

// pathID fills an empty body ID from the URL and rejects a mismatch.
func pathID(r *http.Request, id *string) error {
	p := r.PathValue("id")
	switch strings.TrimSpace(*id) {
	case "", p:
		*id = p
		return nil
	}
	return errors.New("id in body does not match URL")
}
