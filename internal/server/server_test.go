package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgfront/bmax2/internal/maxauth"
	"github.com/dgfront/bmax2/internal/store"
	"github.com/dgfront/bmax2/web"
)

const (
	botToken   = "test-bot-token"
	adminToken = "0123456789abcdef0123456789abcdef"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.ImportDir(ctx, "../../seed"); err != nil {
		t.Fatalf("import seed: %v", err)
	}
	srv, err := New(ctx, Options{
		Store:      st,
		Validator:  maxauth.NewValidator(botToken, time.Hour),
		AdminToken: adminToken,
		Web:        web.Files(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, st
}

func initData(userID int64, startParam string) string {
	fields := map[string]string{
		"auth_date":   strconv.FormatInt(time.Now().Unix(), 10),
		"start_param": startParam,
		"user":        `{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"Тест"}`,
	}
	keys := []string{"auth_date", "start_param", "user"}
	check := make([]string, len(keys))
	pairs := make([]string, len(keys))
	for i, k := range keys {
		check[i] = k + "=" + fields[k]
		pairs[i] = k + "=" + url.PathEscape(fields[k])
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(check, "\n")))
	return strings.Join(pairs, "&") + "&hash=" + hex.EncodeToString(mac.Sum(nil))
}

func do(t *testing.T, method, url string, body io.Reader, header map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

type publicCatalogResp struct {
	Featured   []string `json:"featured"`
	Categories []struct {
		ID            string   `json:"id"`
		Link          string   `json:"link"`
		Organizations []string `json:"organizations"`
		Sections      []struct {
			Title         string   `json:"title"`
			Organizations []string `json:"organizations"`
		} `json:"sections"`
	} `json:"categories"`
	Organizations []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		URL      string `json:"url"`
		Avatar   string `json:"avatar"`
		Subtitle string `json:"subtitle"`
	} `json:"organizations"`
}

func getCatalog(t *testing.T, base string) publicCatalogResp {
	t.Helper()
	resp := do(t, "GET", base+"/api/v1/catalog", nil, map[string]string{InitDataHeader: initData(1, "")})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("catalog status %d", resp.StatusCode)
	}
	var c publicCatalogResp
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPublicAPI(t *testing.T) {
	ts, st := newTestServer(t)

	if resp := do(t, "GET", ts.URL+"/api/v1/catalog", nil, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("catalog without initData: status %d", resp.StatusCode)
	}

	c := getCatalog(t, ts.URL)
	if strings.Join(c.Featured, ",") != "governor,gov,parlament,mfc" {
		t.Fatalf("featured = %v", c.Featured)
	}
	if len(c.Organizations) < 100 || len(c.Categories) != 15 {
		t.Fatalf("got %d organizations, %d categories", len(c.Organizations), len(c.Categories))
	}
	if c.Categories[0].ID != "fuel" || c.Categories[0].Link != "eirc_fuel" {
		t.Fatalf("first category = %+v", c.Categories[0])
	}
	var titles []string
	for _, cat := range c.Categories {
		if cat.ID == "government" {
			for _, s := range cat.Sections {
				titles = append(titles, s.Title)
			}
		}
	}
	if strings.Join(titles, "|") != "Министерства|Комитеты|Службы и управления" {
		t.Fatalf("government sections = %v", titles)
	}
	var avatar string
	for _, o := range c.Organizations {
		if o.ID == "governor" {
			avatar = o.Avatar
		}
	}
	if !strings.HasPrefix(avatar, "/media/gsm.jpg?v=") {
		t.Fatalf("governor avatar = %q", avatar)
	}

	resp := do(t, "GET", ts.URL+avatar, nil, nil)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/jpeg" ||
		!strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("media: status %d, headers %v", resp.StatusCode, resp.Header)
	}

	etag := do(t, "GET", ts.URL+"/api/v1/catalog", nil, map[string]string{InitDataHeader: initData(1, "")}).Header.Get("ETag")
	resp = do(t, "GET", ts.URL+"/api/v1/catalog", nil, map[string]string{
		InitDataHeader: initData(1, ""), "If-None-Match": etag,
	})
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional catalog: status %d", resp.StatusCode)
	}

	for i := 0; i < 2; i++ {
		resp = do(t, "POST", ts.URL+"/api/v1/session", nil, map[string]string{InitDataHeader: initData(7, "category_health")})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("session: status %d", resp.StatusCode)
		}
	}
	stats, err := st.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Users != 1 || stats.Launches != 2 {
		t.Fatalf("stats = %+v", stats)
	}

	resp = do(t, "GET", ts.URL+"/", nil, nil)
	page, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !bytes.Contains(page, []byte("max-web-app.js")) {
		t.Fatalf("index: status %d", resp.StatusCode)
	}
	if resp := do(t, "GET", ts.URL+"/js/app.js", nil, nil); !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/javascript") {
		t.Fatalf("app.js content type %q", resp.Header.Get("Content-Type"))
	}
}

func TestAdminAPI(t *testing.T) {
	ts, _ := newTestServer(t)
	auth := map[string]string{"Authorization": "Bearer " + adminToken}

	if resp := do(t, "GET", ts.URL+"/api/admin/stats", nil, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin without token: status %d", resp.StatusCode)
	}

	jpeg, err := os.ReadFile("../../seed/media/mfc.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if resp := do(t, "PUT", ts.URL+"/api/admin/media/new-logo.jpg", bytes.NewReader(jpeg), auth); resp.StatusCode != http.StatusOK {
		t.Fatalf("upload media: status %d", resp.StatusCode)
	}
	if resp := do(t, "PUT", ts.URL+"/api/admin/media/bad.png", bytes.NewReader(jpeg), auth); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("mismatched media type: status %d", resp.StatusCode)
	}

	org := `{"name":"Новая организация","url":"https://max.ru/new_org","icon":"🏢","avatar":"new-logo.jpg","categories":["health","social"]}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/organizations/new_org", strings.NewReader(org), auth); resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("put organization: status %d: %s", resp.StatusCode, body)
	}

	c := getCatalog(t, ts.URL)
	found := false
	for _, o := range c.Organizations {
		if o.ID == "new_org" {
			found = o.Subtitle == "Здравоохранение" && strings.HasPrefix(o.Avatar, "/media/new-logo.jpg?v=")
		}
	}
	if !found {
		t.Fatal("new organization missing from public catalog")
	}

	dup := `{"name":"Дубль","url":"https://max.ru/new_org","categories":[]}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/organizations/dup_org", strings.NewReader(dup), auth); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate url: status %d", resp.StatusCode)
	}
	badCat := `{"name":"X","url":"https://example.org/","categories":["nope"]}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/organizations/x", strings.NewReader(badCat), auth); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown category: status %d", resp.StatusCode)
	}

	badGroup := `{"name":"Г","url":"https://max.ru/g1","group":"Министерства","categories":["health"]}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/organizations/g1", strings.NewReader(badGroup), auth); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("group outside category: status %d", resp.StatusCode)
	}
	goodGroup := `{"name":"Г","url":"https://max.ru/g1","group":"Министерства","categories":["government"]}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/organizations/g1", strings.NewReader(goodGroup), auth); resp.StatusCode != http.StatusOK {
		t.Fatalf("group in category: status %d", resp.StatusCode)
	}

	if resp := do(t, "PUT", ts.URL+"/api/admin/featured", strings.NewReader(`["new_org","governor"]`), auth); resp.StatusCode != http.StatusOK {
		t.Fatalf("featured: status %d", resp.StatusCode)
	}
	if c := getCatalog(t, ts.URL); len(c.Featured) != 2 || c.Featured[0] != "new_org" {
		t.Fatalf("featured = %v", c.Featured)
	}

	if resp := do(t, "DELETE", ts.URL+"/api/admin/organizations/new_org", nil, auth); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete organization: status %d", resp.StatusCode)
	}
	if resp := do(t, "DELETE", ts.URL+"/api/admin/organizations/new_org", nil, auth); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing organization: status %d", resp.StatusCode)
	}

	// Export, then import the export back unchanged.
	resp := do(t, "GET", ts.URL+"/api/admin/catalog", nil, auth)
	exported, _ := io.ReadAll(resp.Body)
	if resp := do(t, "PUT", ts.URL+"/api/admin/catalog", bytes.NewReader(exported), auth); resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("reimport: status %d: %s", resp.StatusCode, body)
	}

	cat := `{"title":"Парковки","icon":"🅿️","wide":false}`
	if resp := do(t, "PUT", ts.URL+"/api/admin/categories/parking", strings.NewReader(cat), auth); resp.StatusCode != http.StatusOK {
		t.Fatalf("put category: status %d", resp.StatusCode)
	}
	if c := getCatalog(t, ts.URL); c.Categories[len(c.Categories)-1].ID != "parking" {
		t.Fatal("new category is not last")
	}
}
