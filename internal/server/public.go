package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/dgfront/bmax2/internal/catalog"
	"github.com/dgfront/bmax2/internal/store"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	snap := s.snapshot.Load()
	h := w.Header()
	h.Set("ETag", snap.ETag)
	// The catalog is the same for every user, so revalidation by ETag is safe.
	h.Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == snap.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", "application/json; charset=utf-8")
	w.Write(snap.Body)
}

// handleSession records the launch of the mini app by a MAX user.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	data := initDataFrom(r.Context())
	if err := s.store.RecordLaunch(r.Context(), data); err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	s.log.InfoContext(r.Context(), "launch", "user", data.User.ID, "start_param", data.StartParam)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       map[string]any{"id": data.User.ID, "firstName": data.User.FirstName},
		"startParam": data.StartParam,
	})
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !catalog.ValidMediaName(name) {
		http.NotFound(w, r)
		return
	}
	m, err := s.store.GetMedia(r.Context(), name)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.writeStoreError(w, r, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", m.ContentType)
	h.Set("ETag", `"`+m.SHA256[:32]+`"`)
	if r.URL.Query().Get("v") != "" {
		// Versioned URLs change whenever the content changes.
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "public, no-cache")
	}
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(m.Data))
}

// staticFiles serves the embedded mini app with content-hash ETags so clients
// revalidate cheaply and always pick up a new release.
type staticFiles struct {
	files map[string]staticFile
}

type staticFile struct {
	data        []byte
	etag        string
	contentType string
}

func newStaticFiles(fsys fs.FS) (*staticFiles, error) {
	sf := &staticFiles{files: make(map[string]staticFile)}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		ct := mime.TypeByExtension(path.Ext(p))
		if ct == "" {
			ct = http.DetectContentType(data)
		}
		sf.files["/"+p] = staticFile{data: data, etag: `"` + hex.EncodeToString(sum[:16]) + `"`, contentType: ct}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, ok := sf.files["/index.html"]; !ok {
		return nil, errors.New("web: index.html is missing")
	}
	return sf, nil
}

func (sf *staticFiles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	if p == "/" || strings.HasSuffix(p, "/index.html") {
		p = "/index.html"
	}
	f, ok := sf.files[p]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Type", f.contentType)
	h.Set("ETag", f.etag)
	h.Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(f.data))
}
