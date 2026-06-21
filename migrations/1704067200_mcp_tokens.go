package migrations

import (
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/tools/types"
)

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Check if collection already exists
		existing, _ := dao.FindCollectionByNameOrId("_pb_mcp_tokens_")
		if existing != nil {
			return nil
		}

		// Create MCP tokens collection
		mcpTokensCollection := &models.Collection{}
		mcpTokensCollection.MarkAsNew()
		mcpTokensCollection.Id = "_pb_mcp_tokens_"
		mcpTokensCollection.Name = "mcp_tokens"
		mcpTokensCollection.Type = models.CollectionTypeBase
		mcpTokensCollection.System = false
		mcpTokensCollection.ListRule = types.Pointer("")
		mcpTokensCollection.ViewRule = types.Pointer("")
		mcpTokensCollection.CreateRule = types.Pointer("")
		mcpTokensCollection.UpdateRule = types.Pointer("")
		mcpTokensCollection.DeleteRule = types.Pointer("")

		// Define schema fields
		mcpTokensCollection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:   "mcp_token_name",
				Type: schema.FieldTypeText,
				Name: "name",
				Options: &schema.TextOptions{
					Min:     types.Pointer(1),
					Max:     types.Pointer(255),
					Pattern: "",
				},
			},
			&schema.SchemaField{
				Id:   "mcp_token_value",
				Type: schema.FieldTypeText,
				Name: "token",
				Options: &schema.TextOptions{
					Min:     types.Pointer(1),
					Max:     types.Pointer(500),
					Pattern: "",
				},
			},
			&schema.SchemaField{
				Id:   "mcp_token_description",
				Type: schema.FieldTypeText,
				Name: "description",
				Options: &schema.TextOptions{
					Min:     nil,
					Max:     types.Pointer(1000),
					Pattern: "",
				},
			},
			&schema.SchemaField{
				Id:   "mcp_token_active",
				Type: schema.FieldTypeBool,
				Name: "active",
				Options: &schema.BoolOptions{},
			},
			&schema.SchemaField{
				Id:   "mcp_token_expires",
				Type: schema.FieldTypeDate,
				Name: "expiresAt",
				Options: &schema.DateOptions{},
			},
		)

		return dao.SaveCollection(mcpTokensCollection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)
		
		// Find and delete the collection
		collection, _ := dao.FindCollectionByNameOrId("_pb_mcp_tokens_")
		if collection != nil {
			return dao.DeleteCollection(collection)
		}
		
		return nil
	})
}
