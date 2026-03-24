package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create nodes collection
		nodesCollection := core.NewBaseCollection("nodes")
		nodesCollection.Fields.Add(&core.TextField{
			Name:     "path",
			Required: true,
		})
		nodesCollection.AddIndex("idx_nodes_path", true, "path", "")

		// Lock down API rules (nil means deny all)
		nodesCollection.ListRule = nil
		nodesCollection.ViewRule = nil
		nodesCollection.CreateRule = nil
		nodesCollection.UpdateRule = nil
		nodesCollection.DeleteRule = nil

		if err := app.Save(nodesCollection); err != nil {
			return err
		}

		// Look up users collection id
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Create items collection
		itemsCollection := core.NewBaseCollection("items")

		itemsCollection.Fields.Add(&core.RelationField{
			Name:         "node",
			Required:     true,
			CollectionId: nodesCollection.Id,
			MaxSelect:    1,
		})

		itemsCollection.Fields.Add(&core.RelationField{
			Name:         "user",
			Required:     true,
			CollectionId: usersCollection.Id,
			MaxSelect:    1,
		})

		itemsCollection.Fields.Add(&core.SelectField{
			Name:     "type",
			Required: true,
			Values:   []string{"link", "note"},
		})

		itemsCollection.Fields.Add(&core.TextField{
			Name: "title",
			Max:  200,
		})

		itemsCollection.Fields.Add(&core.URLField{
			Name: "url",
		})

		itemsCollection.Fields.Add(&core.TextField{
			Name: "body",
		})

		itemsCollection.Fields.Add(&core.NumberField{
			Name: "sort_order",
		})

		// Lock down API rules
		itemsCollection.ListRule = nil
		itemsCollection.ViewRule = nil
		itemsCollection.CreateRule = nil
		itemsCollection.UpdateRule = nil
		itemsCollection.DeleteRule = nil

		return app.Save(itemsCollection)
	}, func(app core.App) error {
		// Down migration: drop collections
		itemsCollection, err := app.FindCollectionByNameOrId("items")
		if err == nil {
			if err := app.Delete(itemsCollection); err != nil {
				return err
			}
		}

		nodesCollection, err := app.FindCollectionByNameOrId("nodes")
		if err == nil {
			if err := app.Delete(nodesCollection); err != nil {
				return err
			}
		}

		return nil
	})
}
