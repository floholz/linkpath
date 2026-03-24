package handlers

import (
	"github.com/pocketbase/pocketbase/core"
	"linkpath/internal/render"
)

// ItemCard holds a record and its pre-rendered body HTML for use in templates.
type ItemCard struct {
	Item        *core.Record
	CurrentPath string
	BodyHTML    string
	ReadOnly    bool
}

// AncestorGroupData holds item cards for a particular ancestor path.
type AncestorGroupData struct {
	Path      string
	ItemCards []ItemCard
}

// buildItemCard builds an ItemCard from a record, rendering markdown if needed.
func buildItemCard(record *core.Record, currentPath string, readOnly bool) ItemCard {
	var bodyHTML string
	if record.GetString("type") == "note" && record.GetString("body") != "" {
		html, err := render.MarkdownToHTML(record.GetString("body"))
		if err == nil {
			bodyHTML = html
		} else {
			bodyHTML = record.GetString("body")
		}
	}
	return ItemCard{
		Item:        record,
		CurrentPath: currentPath,
		BodyHTML:    bodyHTML,
		ReadOnly:    readOnly,
	}
}
