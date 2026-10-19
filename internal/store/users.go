package store

import (
	"context"

	"github.com/dgfront/bmax2/internal/maxauth"
)

// RecordLaunch upserts the user from validated initData and counts a launch.
func (s *Store) RecordLaunch(ctx context.Context, d maxauth.InitData) error {
	u := d.User
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_users (id, first_name, last_name, username, language_code, last_start_param)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			first_name = excluded.first_name, last_name = excluded.last_name,
			username = excluded.username, language_code = excluded.language_code,
			last_start_param = excluded.last_start_param,
			launches = launches + 1,
			last_seen_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		u.ID, u.FirstName, u.LastName, u.Username, u.LanguageCode, d.StartParam)
	return err
}

type Stats struct {
	Users         int `json:"users"`
	Launches      int `json:"launches"`
	ActiveDay     int `json:"activeLastDay"`
	ActiveWeek    int `json:"activeLastWeek"`
	ActiveMonth   int `json:"activeLastMonth"`
	Organizations int `json:"organizations"`
	Categories    int `json:"categories"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(launches), 0),
			COUNT(*) FILTER (WHERE last_seen_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-1 day')),
			COUNT(*) FILTER (WHERE last_seen_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-7 days')),
			COUNT(*) FILTER (WHERE last_seen_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-30 days')),
			(SELECT COUNT(*) FROM organizations),
			(SELECT COUNT(*) FROM categories)
		FROM app_users`).
		Scan(&st.Users, &st.Launches, &st.ActiveDay, &st.ActiveWeek, &st.ActiveMonth,
			&st.Organizations, &st.Categories)
	return st, err
}
