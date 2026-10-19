package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
)

type Media struct {
	Name        string
	ContentType string
	Data        []byte
	SHA256      string
}

// PutMedia creates or replaces a media file.
func (s *Store) PutMedia(ctx context.Context, name, contentType string, data []byte) error {
	sum := sha256.Sum256(data)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO media (name, content_type, data, sha256) VALUES (?, ?, ?, ?)
		ON CONFLICT (name) DO UPDATE SET
			content_type = excluded.content_type, data = excluded.data, sha256 = excluded.sha256,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		name, contentType, data, hex.EncodeToString(sum[:]))
	return err
}

func (s *Store) GetMedia(ctx context.Context, name string) (Media, error) {
	m := Media{Name: name}
	err := s.db.QueryRowContext(ctx,
		`SELECT content_type, data, sha256 FROM media WHERE name = ?`, name).
		Scan(&m.ContentType, &m.Data, &m.SHA256)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

// DeleteMedia removes a media file; organizations using it lose the avatar.
func (s *Store) DeleteMedia(ctx context.Context, name string) error {
	return s.deleteByID(ctx, `DELETE FROM media WHERE name = ?`, name)
}

// MediaVersions maps every media name to a short content hash.
func (s *Store) MediaVersions(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, substr(sha256, 1, 12) FROM media`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := make(map[string]string)
	for rows.Next() {
		var name, v string
		if err := rows.Scan(&name, &v); err != nil {
			return nil, err
		}
		versions[name] = v
	}
	return versions, rows.Err()
}
