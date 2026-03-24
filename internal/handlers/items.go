package handlers

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"linkpath/internal/middleware"
	"linkpath/internal/pathutil"
	"linkpath/internal/render"
)

func CreateItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		itemType := r.FormValue("type")
		title := r.FormValue("title")
		itemURL := r.FormValue("url")
		body := r.FormValue("body")
		currentPath := r.FormValue("current_path")

		if itemType != "link" && itemType != "note" {
			http.Error(w, "Invalid item type", http.StatusBadRequest)
			return
		}

		node, err := findOrCreateNode(app, pathutil.Normalize(currentPath))
		if err != nil {
			http.Error(w, "Failed to find node", http.StatusInternalServerError)
			return
		}

		collection, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// Assign sort_order = max existing + 1 so new items append to the bottom.
		sortOrder := 0
		existing, err := app.FindRecordsByFilter(
			"items",
			"node = {:nodeId} && user = {:userId}",
			"-sort_order",
			1, 0,
			dbx.Params{"nodeId": node.Id, "userId": user.Id},
		)
		if err == nil && len(existing) > 0 {
			sortOrder = existing[0].GetInt("sort_order") + 1
		}

		record := core.NewRecord(collection)
		record.Set("node", node.Id)
		record.Set("user", user.Id)
		record.Set("type", itemType)
		if itemType == "link" {
			record.Set("title", title)
			record.Set("url", itemURL)
		}
		if itemType == "note" {
			record.Set("body", body)
		}
		record.Set("sort_order", sortOrder)

		if err := app.Save(record); err != nil {
			http.Error(w, "Failed to save item", http.StatusInternalServerError)
			return
		}

		tmpl.RenderPartial(w, "item_card.html", buildItemCard(record, currentPath, false))
	}
}

func GetItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		record, err := app.FindRecordById("items", id)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if record.GetString("user") != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		currentPath := r.URL.Query().Get("path")
		tmpl.RenderPartial(w, "item_card.html", buildItemCard(record, currentPath, false))
	}
}

func EditItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		record, err := app.FindRecordById("items", id)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if record.GetString("user") != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		currentPath := r.URL.Query().Get("path")
		tmpl.RenderPartial(w, "edit_form.html", map[string]any{
			"Item":        record,
			"CurrentPath": currentPath,
		})
	}
}

func UpdateItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		record, err := app.FindRecordById("items", id)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if record.GetString("user") != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		currentPath := r.FormValue("current_path")
		itemType := record.GetString("type")

		if itemType == "link" {
			record.Set("title", r.FormValue("title"))
			record.Set("url", r.FormValue("url"))
		}
		if itemType == "note" {
			record.Set("body", r.FormValue("body"))
		}

		if err := app.Save(record); err != nil {
			http.Error(w, "Failed to update item", http.StatusInternalServerError)
			return
		}

		tmpl.RenderPartial(w, "item_card.html", buildItemCard(record, currentPath, false))
	}
}

func DeleteItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		record, err := app.FindRecordById("items", id)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if record.GetString("user") != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if err := app.Delete(record); err != nil {
			http.Error(w, "Failed to delete item", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func AddFormHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentPath := r.URL.Query().Get("path")
		itemType := r.URL.Query().Get("type")
		if itemType != "link" && itemType != "note" {
			itemType = "link"
		}
		tmpl.RenderPartial(w, "add_form.html", map[string]any{
			"CurrentPath": currentPath,
			"Type":        itemType,
		})
	}
}

func MoveItemHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		record, err := app.FindRecordById("items", id)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if record.GetString("user") != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		direction := r.URL.Query().Get("direction")
		currentPath := r.URL.Query().Get("path")

		// Load all items at this node for this user, sorted by sort_order.
		items, err := app.FindRecordsByFilter(
			"items",
			"node = {:nodeId} && user = {:userId}",
			"+sort_order",
			500, 0,
			dbx.Params{"nodeId": record.GetString("node"), "userId": user.Id},
		)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// Find index of the moved item.
		idx := -1
		for i, it := range items {
			if it.Id == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			http.Error(w, "Item not found in list", http.StatusInternalServerError)
			return
		}

		swapIdx := idx - 1
		if direction == "down" {
			swapIdx = idx + 1
		}

		// Normalize sort_orders and perform the swap.
		if swapIdx >= 0 && swapIdx < len(items) {
			items[idx], items[swapIdx] = items[swapIdx], items[idx]
		}
		for i, it := range items {
			it.Set("sort_order", i)
			if saveErr := app.Save(it); saveErr != nil {
				http.Error(w, "Failed to save order", http.StatusInternalServerError)
				return
			}
		}

		cards := buildItemCards(items, currentPath)
		tmpl.RenderPartial(w, "items_list.html", cards)
	}
}
