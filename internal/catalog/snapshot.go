package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Snapshot is the pre-encoded public catalog served to the mini app.
type Snapshot struct {
	Body []byte
	ETag string
}

type publicCatalog struct {
	Featured      []string         `json:"featured"`
	Categories    []publicCategory `json:"categories"`
	Organizations []publicOrg      `json:"organizations"`
}

type publicCategory struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Wide  bool   `json:"wide"`
	// Link is the organization the tile opens directly, if any.
	Link          string   `json:"link,omitempty"`
	Organizations []string `json:"organizations"`
	// Sections splits Organizations under subheadings when the category
	// defines groups; organizations without a group come last, untitled.
	Sections []publicSection `json:"sections,omitempty"`
}

type publicSection struct {
	Title         string   `json:"title"`
	Organizations []string `json:"organizations"`
}

type publicOrg struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Avatar   string `json:"avatar"`
	Subtitle string `json:"subtitle"`
}

// BuildSnapshot encodes doc for the app. mediaVersions maps media names to a
// content hash used for cache busting; avatars missing from it are dropped.
func BuildSnapshot(doc Document, mediaVersions map[string]string) (*Snapshot, error) {
	titles := make(map[string]string, len(doc.Categories))
	for _, c := range doc.Categories {
		titles[c.ID] = c.Title
	}

	orgs := make([]publicOrg, len(doc.Organizations))
	for i, o := range doc.Organizations {
		p := publicOrg{ID: o.ID, Name: o.Name, URL: o.URL, Icon: o.Icon}
		if v, ok := mediaVersions[o.Avatar]; ok {
			p.Avatar = "/media/" + o.Avatar + "?v=" + v
		}
		if len(o.Categories) > 0 {
			p.Subtitle = titles[o.Categories[0]]
		}
		orgs[i] = p
	}
	col := collate.New(language.Russian, collate.Loose, collate.Numeric)
	keys := make(map[string]string, len(orgs))
	for _, o := range orgs {
		keys[o.ID] = sortKey(o.Name)
	}
	sort.SliceStable(orgs, func(i, j int) bool {
		return col.CompareString(keys[orgs[i].ID], keys[orgs[j].ID]) < 0
	})

	rank := make(map[string]int, len(orgs))
	for i, o := range orgs {
		rank[o.ID] = i
	}
	members := make(map[string][]string, len(doc.Categories))
	groupOf := make(map[string]string, len(doc.Organizations))
	for _, o := range doc.Organizations {
		groupOf[o.ID] = o.Group
		for _, c := range o.Categories {
			members[c] = append(members[c], o.ID)
		}
	}

	cats := make([]publicCategory, len(doc.Categories))
	for i, c := range doc.Categories {
		ids := members[c.ID]
		if ids == nil {
			ids = []string{}
		}
		sort.Slice(ids, func(a, b int) bool { return rank[ids[a]] < rank[ids[b]] })
		cats[i] = publicCategory{
			ID: c.ID, Title: c.Title, Icon: c.Icon, Wide: c.Wide, Link: c.Link,
			Organizations: ids, Sections: sections(c.Groups, ids, groupOf),
		}
	}

	featured := doc.Featured
	if featured == nil {
		featured = []string{}
	}
	body, err := json.Marshal(publicCatalog{Featured: featured, Categories: cats, Organizations: orgs})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	return &Snapshot{Body: body, ETag: `"` + hex.EncodeToString(sum[:16]) + `"`}, nil
}

// sortKey drops leading quotes and symbols so «Name» sorts under N.
func sortKey(name string) string {
	return strings.TrimLeftFunc(name, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// sections groups ids (already sorted) by the category's ordered groups.
func sections(groups, ids []string, groupOf map[string]string) []publicSection {
	if len(groups) == 0 {
		return nil
	}
	byGroup := make(map[string][]string, len(groups))
	var rest []string
	for _, id := range ids {
		if g := groupOf[id]; slices.Contains(groups, g) {
			byGroup[g] = append(byGroup[g], id)
		} else {
			rest = append(rest, id)
		}
	}
	out := make([]publicSection, 0, len(groups)+1)
	for _, g := range groups {
		if len(byGroup[g]) > 0 {
			out = append(out, publicSection{Title: g, Organizations: byGroup[g]})
		}
	}
	if len(rest) > 0 {
		out = append(out, publicSection{Organizations: rest})
	}
	return out
}
