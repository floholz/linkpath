package render

import (
	"html/template"
	"io/fs"
	"net/http"
)

// Templates holds parsed HTML templates.
type Templates struct {
	t *template.Template
}

// New parses all templates from the given filesystem.
// The filesystem should be rooted at the directory containing the templates folder.
func New(fsys fs.FS) (*Templates, error) {
	tmplFS, err := fs.Sub(fsys, "templates")
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	t, err := template.New("").Funcs(funcMap).ParseFS(tmplFS,
		"base.html",
		"landing.html",
		"login.html",
		"register.html",
		"path.html",
		"partials/item_card.html",
		"partials/add_form.html",
		"partials/edit_form.html",
	)
	if err != nil {
		return nil, err
	}

	return &Templates{t: t}, nil
}

// Render executes the named template with the given data, writing to w.
func (tmpl *Templates) Render(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.t.ExecuteTemplate(w, name, data)
}

// RenderPartial executes a partial template (no base layout) with the given data.
func (tmpl *Templates) RenderPartial(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.t.ExecuteTemplate(w, name, data)
}
