package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"linkpath/internal/pathutil"
	"linkpath/internal/render"
)

// PathHandler handles all GET requests — serves landing, home dashboard, or path views.
func PathHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawPath := strings.TrimPrefix(r.URL.Path, "/")

		// Normalize; redirect if changed
		normalized := pathutil.Normalize(rawPath)
		if normalized != rawPath {
			http.Redirect(w, r, "/"+normalized, http.StatusMovedPermanently)
			return
		}

		// Resolve auth from cookie (no middleware on this route)
		user := resolveUser(app, r)

		// Root: landing or home dashboard
		if normalized == "" {
			if user == nil {
				tmpl.Render(w, "landing.html", map[string]any{"User": nil})
				return
			}
			tmpl.Render(w, "path.html", map[string]any{
				"User":            user,
				"CurrentPath":     "",
				"ItemCards":       []ItemCard{},
				"AncestorGroups":  []AncestorGroupData{},
				"DescendantPaths": []string{},
				"IsHome":          true,
			})
			return
		}

		// Non-root paths require auth
		if user == nil {
			http.Redirect(w, r, "/~/login", http.StatusFound)
			return
		}

		// Find-or-create node
		node, err := findOrCreateNode(app, normalized)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Current user's items at this node
		items, err := app.FindRecordsByFilter(
			"items",
			"node = {:nodeId} && user = {:userId}",
			"sort_order,created",
			500, 0,
			dbx.Params{"nodeId": node.Id, "userId": user.Id},
		)
		if err != nil {
			items = []*core.Record{}
		}

		itemCards := make([]ItemCard, 0, len(items))
		for _, item := range items {
			itemCards = append(itemCards, buildItemCard(item, normalized, false))
		}

		// Ancestor paths and their items
		ancestorPaths := pathutil.AncestorPaths(normalized)
		ancestorGroups := make([]AncestorGroupData, 0, len(ancestorPaths))
		for _, ap := range ancestorPaths {
			ancestorNode, err := app.FindFirstRecordByFilter("nodes", "path = {:path}", dbx.Params{"path": ap})
			if err != nil {
				ancestorGroups = append(ancestorGroups, AncestorGroupData{Path: ap})
				continue
			}
			ancestorItems, err := app.FindRecordsByFilter(
				"items",
				"node = {:nodeId} && user = {:userId}",
				"sort_order,created",
				500, 0,
				dbx.Params{"nodeId": ancestorNode.Id, "userId": user.Id},
			)
			if err != nil {
				ancestorItems = []*core.Record{}
			}
			cards := make([]ItemCard, 0, len(ancestorItems))
			for _, item := range ancestorItems {
				cards = append(cards, buildItemCard(item, ap, true))
			}
			ancestorGroups = append(ancestorGroups, AncestorGroupData{Path: ap, ItemCards: cards})
		}

		// Descendant nodes for sidebar tree
		descendants, _ := loadDescendants(app, normalized)
		descPaths := make([]string, 0, len(descendants))
		for _, d := range descendants {
			descPaths = append(descPaths, d.GetString("path"))
		}

		tmpl.Render(w, "path.html", map[string]any{
			"User":            user,
			"CurrentPath":     normalized,
			"CurrentNode":     node,
			"ItemCards":       itemCards,
			"AncestorGroups":  ancestorGroups,
			"DescendantPaths": descPaths,
			"IsHome":          false,
		})
	}
}

func resolveUser(app core.App, r *http.Request) *core.Record {
	cookie, err := r.Cookie("pb_auth")
	if err != nil || cookie.Value == "" {
		return nil
	}
	record, err := app.FindAuthRecordByToken(cookie.Value)
	if err != nil {
		return nil
	}
	return record
}

func findOrCreateNode(app core.App, path string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter("nodes", "path = {:path}", dbx.Params{"path": path})
	if err == nil {
		return record, nil
	}

	collection, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return nil, fmt.Errorf("nodes collection not found: %w", err)
	}

	record = core.NewRecord(collection)
	record.Set("path", path)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	return record, nil
}

func loadDescendants(app core.App, path string) ([]*core.Record, error) {
	prefix := path + "/"
	records, err := app.FindRecordsByFilter(
		"nodes",
		"path ~ {:prefix}",
		"path",
		500, 0,
		dbx.Params{"prefix": prefix},
	)
	if err != nil {
		return nil, err
	}

	result := make([]*core.Record, 0, len(records))
	for _, rec := range records {
		p := rec.GetString("path")
		if strings.HasPrefix(p, prefix) {
			result = append(result, rec)
		}
	}
	return result, nil
}
