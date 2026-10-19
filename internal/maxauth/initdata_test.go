package maxauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

const testToken = "test-bot-token"

// sign builds initData the way MAX does for the given decoded fields.
func sign(t *testing.T, fields map[string]string) string {
	t.Helper()
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	check := make([]string, len(keys))
	pairs := make([]string, len(keys))
	for i, k := range keys {
		check[i] = k + "=" + fields[k]
		pairs[i] = k + "=" + url.QueryEscape(fields[k])
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(testToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(check, "\n")))
	return strings.Join(pairs, "&") + "&hash=" + hex.EncodeToString(mac.Sum(nil))
}

func validator(now time.Time) *Validator {
	v := NewValidator(testToken, time.Hour)
	v.now = func() time.Time { return now }
	return v
}

func TestValidate(t *testing.T) {
	now := time.Unix(1_760_000_000, 0)
	fields := map[string]string{
		"auth_date":   "1759999000",
		"query_id":    "q1",
		"start_param": "category_health",
		"user":        `{"id":42,"first_name":"Алан","username":"alan"}`,
	}
	raw := sign(t, fields)

	got, err := validator(now).Validate(raw)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got.User.ID != 42 || got.User.FirstName != "Алан" || got.StartParam != "category_health" {
		t.Fatalf("unexpected result: %+v", got)
	}

	tests := []struct {
		name string
		raw  string
		now  time.Time
		want error
	}{
		{"tampered", strings.Replace(raw, "category_health", "category_bots", 1), now, ErrSignature},
		{"expired", raw, now.Add(2 * time.Hour), ErrExpired},
		{"future", raw, now.Add(-time.Hour), ErrExpired},
		{"no hash", "auth_date=1&user=%7B%7D", now, ErrMalformed},
		{"empty", "", now, ErrMalformed},
		{"duplicate key", raw + "&query_id=q2", now, ErrMalformed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := validator(tc.now).Validate(tc.raw); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestValidateRequiresUser(t *testing.T) {
	raw := sign(t, map[string]string{"auth_date": "1760000000"})
	if _, err := validator(time.Unix(1_760_000_000, 0)).Validate(raw); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestValidateMillisecondAuthDate(t *testing.T) {
	raw := sign(t, map[string]string{
		"auth_date": "1760000000000",
		"user":      `{"id":1,"first_name":"A"}`,
	})
	if _, err := validator(time.Unix(1_760_000_100, 0)).Validate(raw); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
