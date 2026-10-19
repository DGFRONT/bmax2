package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dgfront/bmax2/internal/catalog"
)

// IsEmpty reports whether the catalog has no categories and organizations.
func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM categories) + (SELECT COUNT(*) FROM organizations)`).Scan(&n)
	return n == 0, err
}

// LoadDocument reads the whole catalog.
func (s *Store) LoadDocument(ctx context.Context) (catalog.Document, error) {
	doc := catalog.Document{
		Categories:    []catalog.Category{},
		Organizations: []catalog.Organization{},
		Featured:      []string{},
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, icon, wide FROM categories ORDER BY position, id`)
	if err != nil {
		return doc, err
	}
	for rows.Next() {
		var c catalog.Category
		if err := rows.Scan(&c.ID, &c.Title, &c.Icon, &c.Wide); err != nil {
			rows.Close()
			return doc, err
		}
		doc.Categories = append(doc.Categories, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return doc, err
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT id, name, url, icon, COALESCE(avatar, '') FROM organizations ORDER BY id`)
	if err != nil {
		return doc, err
	}
	index := make(map[string]int)
	for rows.Next() {
		o := catalog.Organization{Categories: []string{}}
		if err := rows.Scan(&o.ID, &o.Name, &o.URL, &o.Icon, &o.Avatar); err != nil {
			rows.Close()
			return doc, err
		}
		index[o.ID] = len(doc.Organizations)
		doc.Organizations = append(doc.Organizations, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return doc, err
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT organization_id, category_id FROM organization_categories ORDER BY organization_id, position`)
	if err != nil {
		return doc, err
	}
	for rows.Next() {
		var orgID, catID string
		if err := rows.Scan(&orgID, &catID); err != nil {
			rows.Close()
			return doc, err
		}
		o := &doc.Organizations[index[orgID]]
		o.Categories = append(o.Categories, catID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return doc, err
	}

	rows, err = s.db.QueryContext(ctx, `SELECT organization_id FROM featured ORDER BY position`)
	if err != nil {
		return doc, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return doc, err
		}
		doc.Featured = append(doc.Featured, id)
	}
	return doc, rows.Err()
}

// ReplaceDocument atomically replaces the catalog with doc. Media files are
// kept; every avatar must reference an existing media file.
func (s *Store) ReplaceDocument(ctx context.Context, doc catalog.Document) error {
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		for _, q := range []string{
			`DELETE FROM featured`,
			`DELETE FROM organization_categories`,
			`DELETE FROM organizations`,
			`DELETE FROM categories`,
		} {
			if _, err := tx.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		for i, c := range doc.Categories {
			if err := putCategory(ctx, tx, c, i); err != nil {
				return err
			}
		}
		for _, o := range doc.Organizations {
			if err := putOrganization(ctx, tx, o); err != nil {
				return err
			}
		}
		return putFeatured(ctx, tx, doc.Featured)
	})
}

// PutCategory creates or updates a category. New categories go last.
func (s *Store) PutCategory(ctx context.Context, c catalog.Category) error {
	if err := c.Normalize(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		var pos int
		err := tx.QueryRowContext(ctx, `SELECT position FROM categories WHERE id = ?`, c.ID).Scan(&pos)
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM categories`).Scan(&pos)
		}
		if err != nil {
			return err
		}
		return putCategory(ctx, tx, c, pos)
	})
}

// OrderCategories sets the display order; ids must list every category once.
func (s *Store) OrderCategories(ctx context.Context, ids []string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		var total int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories`).Scan(&total); err != nil {
			return err
		}
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			seen[id] = true
		}
		if len(seen) != len(ids) || len(ids) != total {
			return fmt.Errorf("%w: order must list each of the %d categories exactly once", ErrInvalid, total)
		}
		for i, id := range ids {
			res, err := tx.ExecContext(ctx, `UPDATE categories SET position = ? WHERE id = ?`, i, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return fmt.Errorf("%w: unknown category %q", ErrInvalid, id)
			}
		}
		return nil
	})
}

func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	return s.deleteByID(ctx, `DELETE FROM categories WHERE id = ?`, id)
}

// PutOrganization creates or replaces an organization and its categories.
func (s *Store) PutOrganization(ctx context.Context, o catalog.Organization) error {
	if err := o.Normalize(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		var other string
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM organizations WHERE url = ? AND id <> ?`, o.URL, o.ID).Scan(&other)
		if err == nil {
			return fmt.Errorf("%w: url is already used by %q", ErrInvalid, other)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		for _, c := range o.Categories {
			if err := mustExist(ctx, tx, `SELECT 1 FROM categories WHERE id = ?`, c, "unknown category %q"); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM organization_categories WHERE organization_id = ?`, o.ID); err != nil {
			return err
		}
		return putOrganization(ctx, tx, o)
	})
}

func (s *Store) DeleteOrganization(ctx context.Context, id string) error {
	return s.deleteByID(ctx, `DELETE FROM organizations WHERE id = ?`, id)
}

// SetFeatured replaces the featured row with ids in display order.
func (s *Store) SetFeatured(ctx context.Context, ids []string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM featured`); err != nil {
			return err
		}
		return putFeatured(ctx, tx, ids)
	})
}

func (s *Store) deleteByID(ctx context.Context, query, id string) error {
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func putCategory(ctx context.Context, tx *sql.Tx, c catalog.Category, pos int) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO categories (id, title, icon, wide, position) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			title = excluded.title, icon = excluded.icon, wide = excluded.wide,
			position = excluded.position,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		c.ID, c.Title, c.Icon, c.Wide, pos)
	return err
}

func putOrganization(ctx context.Context, tx *sql.Tx, o catalog.Organization) error {
	var avatar any
	if o.Avatar != "" {
		if err := mustExist(ctx, tx, `SELECT 1 FROM media WHERE name = ?`, o.Avatar, "media %q is not uploaded"); err != nil {
			return err
		}
		avatar = o.Avatar
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (id, name, url, icon, avatar) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name, url = excluded.url, icon = excluded.icon,
			avatar = excluded.avatar,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		o.ID, o.Name, o.URL, o.Icon, avatar)
	if err != nil {
		return err
	}
	for i, c := range o.Categories {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO organization_categories (organization_id, category_id, position) VALUES (?, ?, ?)`,
			o.ID, c, i); err != nil {
			return err
		}
	}
	return nil
}

func putFeatured(ctx context.Context, tx *sql.Tx, ids []string) error {
	seen := make(map[string]bool, len(ids))
	for i, id := range ids {
		if seen[id] {
			return fmt.Errorf("%w: featured: %q listed twice", ErrInvalid, id)
		}
		seen[id] = true
		if err := mustExist(ctx, tx, `SELECT 1 FROM organizations WHERE id = ?`, id, "featured: unknown organization %q"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO featured (organization_id, position) VALUES (?, ?)`, id, i); err != nil {
			return err
		}
	}
	return nil
}

// mustExist returns an ErrInvalid error formatted with msg and arg when query
// finds no row.
func mustExist(ctx context.Context, tx *sql.Tx, query string, arg any, msg string) error {
	var one int
	err := tx.QueryRowContext(ctx, query, arg).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: "+msg, ErrInvalid, arg)
	}
	return err
}
