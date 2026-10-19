// Package catalog defines the editable catalog document (used by import,
// export and the admin API) and the read-only snapshot served to the app.
package catalog

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Document is the full editable catalog. Category order and featured order
// follow array order.
type Document struct {
	Categories    []Category     `json:"categories"`
	Organizations []Organization `json:"organizations"`
	Featured      []string       `json:"featured"`
}

type Category struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	// Wide categories span the full row of the grid.
	Wide bool `json:"wide"`
	// Link, when set, is the ID of an organization the tile opens directly
	// instead of showing a list.
	Link string `json:"link"`
	// Groups are ordered subheadings of the category list; organizations
	// choose one with their Group field.
	Groups []string `json:"groups"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// URL is a max.ru link (channel, chat, bot, user) opened inside MAX, or
	// any other https link opened in the browser.
	URL  string `json:"url"`
	Icon string `json:"icon"`
	// Avatar is the name of an uploaded media file, empty for none.
	Avatar string `json:"avatar"`
	// Group is the subheading the organization is listed under in categories
	// that define it in Groups.
	Group string `json:"group"`
	// Categories lists category IDs; the first one is shown as the subtitle.
	Categories []string `json:"categories"`
}

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	mediaPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}\.(jpg|png)$`)
)

// ValidID reports whether s is a valid category or organization ID.
func ValidID(s string) bool { return idPattern.MatchString(s) }

// ValidMediaName reports whether s is a valid media file name.
func ValidMediaName(s string) bool { return mediaPattern.MatchString(s) }

func (c *Category) Normalize() error {
	c.Title = strings.TrimSpace(c.Title)
	c.Icon = strings.TrimSpace(c.Icon)
	c.Link = strings.TrimSpace(c.Link)
	if c.Groups == nil {
		c.Groups = []string{}
	}
	switch {
	case !ValidID(c.ID):
		return fmt.Errorf("category %q: id must match %s", c.ID, idPattern)
	case c.Title == "" || utf8.RuneCountInString(c.Title) > 120:
		return fmt.Errorf("category %q: title must be 1-120 characters", c.ID)
	case utf8.RuneCountInString(c.Icon) > 16:
		return fmt.Errorf("category %q: icon is too long", c.ID)
	case c.Link != "" && !ValidID(c.Link):
		return fmt.Errorf("category %q: link must be an organization id", c.ID)
	case c.Link != "" && len(c.Groups) > 0:
		return fmt.Errorf("category %q: a link tile cannot have groups", c.ID)
	}
	seen := make(map[string]bool, len(c.Groups))
	for i, g := range c.Groups {
		g = strings.TrimSpace(g)
		if g == "" || utf8.RuneCountInString(g) > 80 || seen[g] {
			return fmt.Errorf("category %q: groups must be unique, 1-80 characters", c.ID)
		}
		seen[g] = true
		c.Groups[i] = g
	}
	return nil
}

func (o *Organization) Normalize() error {
	o.Name = strings.TrimSpace(o.Name)
	o.URL = strings.TrimSpace(o.URL)
	o.Icon = strings.TrimSpace(o.Icon)
	o.Group = strings.TrimSpace(o.Group)
	if o.Categories == nil {
		o.Categories = []string{}
	}
	switch {
	case !ValidID(o.ID):
		return fmt.Errorf("organization %q: id must match %s", o.ID, idPattern)
	case o.Name == "" || utf8.RuneCountInString(o.Name) > 200:
		return fmt.Errorf("organization %q: name must be 1-200 characters", o.ID)
	case utf8.RuneCountInString(o.Icon) > 16:
		return fmt.Errorf("organization %q: icon is too long", o.ID)
	case utf8.RuneCountInString(o.Group) > 80:
		return fmt.Errorf("organization %q: group is too long", o.ID)
	case o.Avatar != "" && !ValidMediaName(o.Avatar):
		return fmt.Errorf("organization %q: invalid avatar name %q", o.ID, o.Avatar)
	}
	if err := checkURL(o.URL); err != nil {
		return fmt.Errorf("organization %q: %w", o.ID, err)
	}
	seen := make(map[string]bool, len(o.Categories))
	for _, c := range o.Categories {
		if seen[c] {
			return fmt.Errorf("organization %q: category %q listed twice", o.ID, c)
		}
		seen[c] = true
	}
	return nil
}

func checkURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || len(raw) > 1024 {
		return errors.New("url must be an absolute https link")
	}
	return nil
}

// Validate normalizes every entry and checks cross references.
func (d *Document) Validate() error {
	cats := make(map[string]*Category, len(d.Categories))
	for i := range d.Categories {
		c := &d.Categories[i]
		if err := c.Normalize(); err != nil {
			return err
		}
		if cats[c.ID] != nil {
			return fmt.Errorf("category %q is duplicated", c.ID)
		}
		cats[c.ID] = c
	}

	orgs := make(map[string]bool, len(d.Organizations))
	urls := make(map[string]string, len(d.Organizations))
	for i := range d.Organizations {
		o := &d.Organizations[i]
		if err := o.Normalize(); err != nil {
			return err
		}
		if orgs[o.ID] {
			return fmt.Errorf("organization %q is duplicated", o.ID)
		}
		orgs[o.ID] = true
		if other, dup := urls[o.URL]; dup {
			return fmt.Errorf("organizations %q and %q share url %s", other, o.ID, o.URL)
		}
		urls[o.URL] = o.ID
		groupOK := o.Group == ""
		for _, c := range o.Categories {
			cat := cats[c]
			if cat == nil {
				return fmt.Errorf("organization %q: unknown category %q", o.ID, c)
			}
			groupOK = groupOK || slices.Contains(cat.Groups, o.Group)
		}
		if !groupOK {
			return fmt.Errorf("organization %q: group %q is not defined in its categories", o.ID, o.Group)
		}
	}

	for _, c := range d.Categories {
		if c.Link != "" && !orgs[c.Link] {
			return fmt.Errorf("category %q: link to unknown organization %q", c.ID, c.Link)
		}
	}

	if d.Featured == nil {
		d.Featured = []string{}
	}
	featured := make(map[string]bool, len(d.Featured))
	for _, id := range d.Featured {
		if !orgs[id] {
			return fmt.Errorf("featured: unknown organization %q", id)
		}
		if featured[id] {
			return fmt.Errorf("featured: %q listed twice", id)
		}
		featured[id] = true
	}
	return nil
}
