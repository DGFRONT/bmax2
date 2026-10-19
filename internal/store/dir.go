package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dgfront/bmax2/internal/catalog"
)

// A catalog directory holds catalog.json and a media/ folder with the files
// referenced by organization avatars.
const (
	catalogFile = "catalog.json"
	mediaDir    = "media"
)

// MediaContentType returns the content type for an allowed media name.
func MediaContentType(name string) string {
	switch filepath.Ext(name) {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	}
	return ""
}

// ImportDir uploads every media file from dir/media and replaces the catalog
// with dir/catalog.json.
func (s *Store) ImportDir(ctx context.Context, dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, catalogFile))
	if err != nil {
		return err
	}
	var doc catalog.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s: %w", catalogFile, err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, mediaDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !catalog.ValidMediaName(name) {
			return fmt.Errorf("media/%s: name must be lowercase latin with .jpg, .png or .webp", name)
		}
		data, err := os.ReadFile(filepath.Join(dir, mediaDir, name))
		if err != nil {
			return err
		}
		if err := s.PutMedia(ctx, name, MediaContentType(name), data); err != nil {
			return err
		}
	}
	return s.ReplaceDocument(ctx, doc)
}

// ExportDir writes the catalog and all media files into dir.
func (s *Store) ExportDir(ctx context.Context, dir string) error {
	doc, err := s.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, mediaDir), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, catalogFile), append(raw, '\n'), 0o644); err != nil {
		return err
	}

	versions, err := s.MediaVersions(ctx)
	if err != nil {
		return err
	}
	for name := range versions {
		m, err := s.GetMedia(ctx, name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, mediaDir, name), m.Data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
