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
	"github.com/dgfront/bmax2/internal/media"
)

// A catalog directory holds catalog.json and a media/ folder with the files
// referenced by organization avatars.
const (
	catalogFile = "catalog.json"
	mediaDir    = "media"
)

// ImportDir uploads every media file from dir/media and replaces the catalog
// with dir/catalog.json. Images are cropped and scaled to avatar size. An
// organization without an avatar gets media/<organization id>.jpg (or .png)
// when such a file exists.
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
	files := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !catalog.ValidMediaName(name) {
			return fmt.Errorf("media/%s: name must be lowercase latin with .jpg or .png", name)
		}
		data, err := os.ReadFile(filepath.Join(dir, mediaDir, name))
		if err != nil {
			return err
		}
		if data, err = media.Prepare(name, data); err != nil {
			return fmt.Errorf("media/%s: %w", name, err)
		}
		if err := s.PutMedia(ctx, name, media.ContentType(name), data); err != nil {
			return err
		}
		files[name] = true
	}
	for i := range doc.Organizations {
		o := &doc.Organizations[i]
		for _, ext := range []string{".jpg", ".png"} {
			if o.Avatar == "" && files[o.ID+ext] {
				o.Avatar = o.ID + ext
			}
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
