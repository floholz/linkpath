package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

// Templates holds per-page template sets and a shared partial set.
type Templates struct {
	pages    map[string]*template.Template
	partials *template.Template
}

var funcMap = template.FuncMap{
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
}

// New builds the template sets from the given filesystem.
// Each page gets its own set (base + page) so {{define "content"}} blocks don't
// overwrite each other across pages.
func New(fsys fs.FS) (*Templates, error) {
	tmplFS, err := fs.Sub(fsys, "templates")
	if err != nil {
		return nil, err
	}

	// Shared partial set — used for HTMX swap responses.
	partials, err := template.New("").Funcs(funcMap).ParseFS(tmplFS,
		"partials/item_card.html",
		"partials/items_list.html",
		"partials/add_form.html",
		"partials/edit_form.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}

	// Per-page sets: base.html + page file (+ item_card partial for path.html
	// which calls {{template "item_card_inner" .}} inline).
	pageFiles := map[string][]string{
		"landing.html":  {"base.html", "landing.html"},
		"login.html":    {"base.html", "login.html"},
		"register.html": {"base.html", "register.html"},
		"path.html":     {"base.html", "path.html", "partials/item_card.html", "partials/items_list.html"},
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for name, files := range pageFiles {
		t, err := template.New("").Funcs(funcMap).ParseFS(tmplFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		pages[name] = t
	}

	return &Templates{pages: pages, partials: partials}, nil
}

// Render executes the named page template using the "base" layout.
func (tmpl *Templates) Render(w http.ResponseWriter, name string, data any) error {
	t, ok := tmpl.pages[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return fmt.Errorf("template not found: %s", name)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, "base", data)
}

// RenderPartial executes a named partial template (no base layout).
func (tmpl *Templates) RenderPartial(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.partials.ExecuteTemplate(w, name, data)
}
