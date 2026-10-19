package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	valid := func() Document {
		return Document{
			Categories: []Category{{ID: "health", Title: "Здравоохранение"}},
			Organizations: []Organization{
				{ID: "a", Name: "A", URL: "https://max.ru/a", Categories: []string{"health"}},
				{ID: "b", Name: "B", URL: "https://eirc15.ru/"},
			},
			Featured: []string{"a"},
		}
	}
	d := valid()
	if err := d.Validate(); err != nil {
		t.Fatalf("valid document: %v", err)
	}

	tests := map[string]func(*Document){
		"bad id":           func(d *Document) { d.Organizations[0].ID = "Bad ID" },
		"http url":         func(d *Document) { d.Organizations[0].URL = "http://max.ru/a" },
		"duplicate url":    func(d *Document) { d.Organizations[1].URL = d.Organizations[0].URL },
		"unknown category": func(d *Document) { d.Organizations[0].Categories = []string{"nope"} },
		"unknown featured": func(d *Document) { d.Featured = []string{"zzz"} },
		"empty title":      func(d *Document) { d.Categories[0].Title = " " },
		"bad avatar":       func(d *Document) { d.Organizations[0].Avatar = "../etc/passwd" },
		"unknown link":     func(d *Document) { d.Categories[0].Link = "zzz" },
		"undefined group":  func(d *Document) { d.Organizations[0].Group = "Министерства" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			d := valid()
			mutate(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestBuildSnapshotOrdering(t *testing.T) {
	doc := Document{
		Categories: []Category{{ID: "c", Title: "Категория"}},
		Organizations: []Organization{
			{ID: "z", Name: "Ясли", URL: "https://max.ru/z", Categories: []string{"c"}},
			{ID: "q", Name: "«Альфа»", URL: "https://max.ru/q", Categories: []string{"c"}},
			{ID: "n10", Name: "Школа №10", URL: "https://max.ru/n10", Categories: []string{"c"}},
			{ID: "n9", Name: "Школа №9", URL: "https://max.ru/n9", Categories: []string{"c"}},
		},
	}
	snap, err := BuildSnapshot(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got publicCatalog
	if err := json.Unmarshal(snap.Body, &got); err != nil {
		t.Fatal(err)
	}
	order := strings.Join(got.Categories[0].Organizations, ",")
	if order != "q,n9,n10,z" {
		t.Fatalf("order = %s", order)
	}
	if got.Organizations[0].Subtitle != "Категория" {
		t.Fatalf("subtitle = %q", got.Organizations[0].Subtitle)
	}
}
