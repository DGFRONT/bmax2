// Package maxauth validates the launch parameters (initData) that MAX passes
// to a mini app, following https://dev.max.ru/docs/webapps/validation:
//
//	secret_key = HMAC_SHA256(key="WebAppData", msg=bot_token)
//	hash       = hex(HMAC_SHA256(key=secret_key, msg=data_check_string))
//
// where data_check_string is every "key=value" pair except hash, with values
// percent-decoded, sorted by key and joined with "\n".
package maxauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// maxInitDataLen bounds the accepted input; real initData is well under 4 KB.
const maxInitDataLen = 16 << 10

// clockSkew tolerates an auth_date slightly in the future.
const clockSkew = 30 * time.Second

var (
	ErrMalformed = errors.New("initData is malformed")
	ErrSignature = errors.New("initData signature mismatch")
	ErrExpired   = errors.New("initData is expired")
)

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

type InitData struct {
	User       User
	StartParam string
	QueryID    string
	AuthDate   time.Time
}

type Validator struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewValidator(botToken string, ttl time.Duration) *Validator {
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	return &Validator{secret: mac.Sum(nil), ttl: ttl, now: time.Now}
}

// Validate checks the signature and age of raw initData and returns its
// parsed content.
func (v *Validator) Validate(raw string) (InitData, error) {
	if raw == "" || len(raw) > maxInitDataLen {
		return InitData{}, ErrMalformed
	}

	fields := make(map[string]string)
	for _, pair := range strings.Split(raw, "&") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return InitData{}, ErrMalformed
		}
		if _, dup := fields[key]; dup {
			return InitData{}, ErrMalformed
		}
		// PathUnescape keeps "+" as is, matching decodeURIComponent.
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return InitData{}, ErrMalformed
		}
		fields[key] = decoded
	}

	hash, err := hex.DecodeString(fields["hash"])
	if err != nil || len(hash) != sha256.Size {
		return InitData{}, ErrMalformed
	}
	delete(fields, "hash")

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(fields[k])
	}

	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(b.String()))
	if !hmac.Equal(mac.Sum(nil), hash) {
		return InitData{}, ErrSignature
	}

	authUnix, err := strconv.ParseInt(fields["auth_date"], 10, 64)
	if err != nil {
		return InitData{}, ErrMalformed
	}
	// MAX may send auth_date in milliseconds; normalise to seconds.
	if authUnix > 1e12 {
		authUnix /= 1000
	}
	authDate := time.Unix(authUnix, 0)
	age := v.now().Sub(authDate)
	if age > v.ttl || age < -clockSkew {
		return InitData{}, ErrExpired
	}

	var user User
	if err := json.Unmarshal([]byte(fields["user"]), &user); err != nil || user.ID <= 0 {
		return InitData{}, ErrMalformed
	}

	return InitData{
		User:       user,
		StartParam: fields["start_param"],
		QueryID:    fields["query_id"],
		AuthDate:   authDate,
	}, nil
}
