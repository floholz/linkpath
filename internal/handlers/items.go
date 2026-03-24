package handlers

import (
	"net/http"

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

		record := core.NewRecord(collection)
		record.Set("node", node.Id)
		record.Set("user", user.Id)
		record.Set("type", itemType)
		record.Set("title", title)
		if itemType == "link" {
			record.Set("url", itemURL)
		}
		if itemType == "note" {
			record.Set("body", body)
		}
		record.Set("sort_order", 0)

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

		title := r.FormValue("title")
		itemURL := r.FormValue("url")
		body := r.FormValue("body")
		currentPath := r.FormValue("current_path")

		record.Set("title", title)
		itemType := record.GetString("type")
		if itemType == "link" {
			record.Set("url", itemURL)
		}
		if itemType == "note" {
			record.Set("body", body)
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
