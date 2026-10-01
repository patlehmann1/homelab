package main

import (
	"bytes"
	"html/template"
	"testing"
)

func TestTemplatesRender(t *testing.T) {
	parts := template.Must(template.New("").Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/components.html"))
	snap := NewStore().Snapshot("", "")

	for _, n := range nav {
		page := template.Must(template.Must(parts.Clone()).ParseFS(assets, "templates/pages/"+n.Key+".html"))
		for _, name := range []string{"layout", "swap"} {
			if err := page.ExecuteTemplate(&bytes.Buffer{}, name, view{Page: n.Key, Snapshot: snap}); err != nil {
				t.Errorf("%s/%s: %v", n.Key, name, err)
			}
		}
	}
	for name := range partials {
		if err := parts.ExecuteTemplate(&bytes.Buffer{}, name, view{Snapshot: snap}); err != nil {
			t.Errorf("partial %s: %v", name, err)
		}
	}
}
