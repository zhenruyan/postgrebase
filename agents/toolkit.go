package agents

import (
	"errors"
	"sort"
	"strings"

	"github.com/spf13/cast"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/forms"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/tools/dbutils"
	"github.com/zhenruyan/postgrebase/tools/search"
	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/zhenruyan/postgrebase/tools/types"
)

// ToolSpec describes a registered embedded agent tool.
//
// Per proposal §8.1, a registry entry carries at least: name, description,
// input schema, executor, required permissions and audit category. The
// executor is stored separately in the registry; the remaining control
// metadata (category/risk/audit/approval) lives here.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`

	// Category is either "read" or "write" (proposal §8.3).
	Category string `json:"category"`
	// Risk is one of "low", "medium", "high".
	Risk string `json:"risk"`
	// AuditCategory groups the tool for audit logging (e.g. "data", "schema").
	AuditCategory string `json:"auditCategory"`
	// RequiresApproval marks write operations that must be explicitly
	// authorized before execution.
	RequiresApproval bool `json:"requiresApproval"`
}

// toolMetadata holds the static control metadata applied to each tool spec.
type toolMetadata struct {
	Category         string
	Risk             string
	AuditCategory    string
	RequiresApproval bool
}

// toolMetadataTable maps tool names to their control metadata (proposal §8.1/§8.3).
// Read tools are allowed by default within the project boundary; write tools
// require explicit authorization.
var toolMetadataTable = map[string]toolMetadata{
	"data.query":          {Category: "read", Risk: "low", AuditCategory: "data"},
	"data.get":            {Category: "read", Risk: "low", AuditCategory: "data"},
	"dataset.preview":     {Category: "read", Risk: "low", AuditCategory: "data"},
	"data.insert":         {Category: "write", Risk: "medium", AuditCategory: "data", RequiresApproval: true},
	"data.bulk_insert":    {Category: "write", Risk: "high", AuditCategory: "data", RequiresApproval: true},
	"data.update":         {Category: "write", Risk: "medium", AuditCategory: "data", RequiresApproval: true},
	"data.delete":         {Category: "write", Risk: "high", AuditCategory: "data", RequiresApproval: true},
	"schema.create_table": {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
	"schema.add_field":    {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
	"schema.update_field": {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
	"schema.drop_field":   {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
	"schema.create_index": {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
	"schema.set_relation": {Category: "write", Risk: "high", AuditCategory: "schema", RequiresApproval: true},
}

// applyToolMetadata sets control metadata on a spec from the static table.
// Unknown tools default to a conservative write/high classification so new
// tools are never silently auto-approved.
func applyToolMetadata(spec *ToolSpec) {
	meta, ok := toolMetadataTable[spec.Name]
	if !ok {
		spec.Category = "write"
		spec.Risk = "high"
		spec.AuditCategory = "unknown"
		spec.RequiresApproval = true
		return
	}
	spec.Category = meta.Category
	spec.Risk = meta.Risk
	spec.AuditCategory = meta.AuditCategory
	spec.RequiresApproval = meta.RequiresApproval
}

// ToolExecutionResult is the normalized result returned by a tool executor.
type ToolExecutionResult struct {
	Status  string     `json:"status"`
	Message string     `json:"message,omitempty"`
	Data    any        `json:"data,omitempty"`
	Chart   *ChartHint `json:"chart,omitempty"`
}

// ChartHint is a recommended visualization for a query result (proposal §10.1).
// The frontend uses it to render a chart alongside the raw table; the agent
// itself does not render charts.
type ChartHint struct {
	// Type is one of: table, line, bar, pie, metric.
	Type string `json:"type"`
	// XField is the field to use for the category/time axis.
	XField string `json:"xField,omitempty"`
	// YFields are the numeric series fields.
	YFields []string `json:"yFields,omitempty"`
}

// ToolExecutor executes a tool call.
type ToolExecutor func(args map[string]any) (*ToolExecutionResult, error)

// ToolRegistry holds the static tool list exposed to the agent surface.
type ToolRegistry struct {
	tools map[string]ToolSpec
	execs map[string]ToolExecutor
}

// NewToolRegistry creates a registry with the platform's initial tool set.
func NewToolRegistry() *ToolRegistry {
	tools := []ToolSpec{
		{
			Name:        "schema.create_table",
			Description: "Create a table definition from structured fields.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":     map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"displayName": map[string]any{"type": "string"},
					"fields": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":       map[string]any{"type": "string"},
								"name":     map[string]any{"type": "string"},
								"type":     map[string]any{"type": "string"},
								"remark":   map[string]any{"type": "string"},
								"required": map[string]any{"type": "boolean"},
								"options":  map[string]any{"type": "object"},
							},
						},
					},
				},
				"required": []string{"project", "name"},
			},
		},
		{
			Name:        "data.query",
			Description: "Query project data using structured filters and pagination.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"filter":     map[string]any{"type": "string"},
					"sort":       map[string]any{"type": "string"},
					"page":       map[string]any{"type": "integer"},
					"perPage":    map[string]any{"type": "integer"},
					"skipTotal":  map[string]any{"type": "boolean"},
				},
				"required": []string{"project", "collection"},
			},
		},
		{
			Name:        "data.insert",
			Description: "Insert a record into a project-scoped collection.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"data":       map[string]any{"type": "object"},
				},
				"required": []string{"project", "collection", "data"},
			},
		},
		{
			Name:        "data.bulk_insert",
			Description: "Insert multiple records into a project-scoped collection.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"rows": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "object"},
					},
				},
				"required": []string{"project", "collection", "rows"},
			},
		},
		{
			Name:        "data.get",
			Description: "Fetch a single record from a project-scoped collection.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"id":         map[string]any{"type": "string"},
				},
				"required": []string{"project", "collection", "id"},
			},
		},
		{
			Name:        "data.update",
			Description: "Update a record in a project-scoped collection.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"id":         map[string]any{"type": "string"},
					"data":       map[string]any{"type": "object"},
				},
				"required": []string{"project", "collection", "id", "data"},
			},
		},
		{
			Name:        "data.delete",
			Description: "Delete a record from a project-scoped collection.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"id":         map[string]any{"type": "string"},
				},
				"required": []string{"project", "collection", "id"},
			},
		},
		{
			Name:        "dataset.preview",
			Description: "Preview a dataset snapshot for the current project.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
				},
				"required": []string{"project", "collection"},
			},
		},
		{
			Name:        "schema.add_field",
			Description: "Add a field to an existing table definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"field":      map[string]any{"type": "object"},
				},
				"required": []string{"project", "collection", "field"},
			},
		},
		{
			Name:        "schema.update_field",
			Description: "Update a field in an existing table definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"field":      map[string]any{"type": "object"},
				},
				"required": []string{"project", "collection", "field"},
			},
		},
		{
			Name:        "schema.drop_field",
			Description: "Drop a field from an existing table definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"field":      map[string]any{"type": "object"},
				},
				"required": []string{"project", "collection", "field"},
			},
		},
		{
			Name:        "schema.create_index",
			Description: "Create or replace a collection index definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":    map[string]any{"type": "string"},
					"collection": map[string]any{"type": "string"},
					"index":      map[string]any{"type": "string"},
				},
				"required": []string{"project", "collection", "index"},
			},
		},
		{
			Name:        "schema.set_relation",
			Description: "Create or update a relation field definition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":       map[string]any{"type": "string"},
					"collection":    map[string]any{"type": "string"},
					"field":         map[string]any{"type": "object"},
					"relation":      map[string]any{"type": "string"},
					"cascadeDelete": map[string]any{"type": "boolean"},
					"minSelect":     map[string]any{"type": "integer"},
					"maxSelect":     map[string]any{"type": "integer"},
					"displayFields": map[string]any{"type": "array"},
				},
				"required": []string{"project", "collection", "field", "relation"},
			},
		},
	}

	reg := &ToolRegistry{
		tools: map[string]ToolSpec{},
		execs: map[string]ToolExecutor{},
	}
	for _, tool := range tools {
		applyToolMetadata(&tool)
		reg.tools[tool.Name] = tool
	}

	return reg
}

func (r *ToolRegistry) registered(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// List returns tools sorted by name.
func (r *ToolRegistry) List() []ToolSpec {
	if r == nil {
		return nil
	}

	result := make([]ToolSpec, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// Get returns a tool by name.
func (r *ToolRegistry) Get(name string) (ToolSpec, bool) {
	if r == nil {
		return ToolSpec{}, false
	}
	tool, ok := r.tools[name]
	return tool, ok
}

// SetExecutor registers an executor for the specified tool.
func (r *ToolRegistry) SetExecutor(name string, exec ToolExecutor) {
	if r == nil {
		return
	}
	if r.execs == nil {
		r.execs = map[string]ToolExecutor{}
	}
	r.execs[name] = exec
}

// executor returns the executor registered for a tool, if any.
func (r *ToolRegistry) executor(name string) (ToolExecutor, bool) {
	if r == nil {
		return nil, false
	}
	exec, ok := r.execs[name]
	return exec, ok
}

// Execute runs a registered tool.
func (r *ToolRegistry) Execute(name string, args map[string]any) (*ToolExecutionResult, error) {
	if r == nil {
		return nil, errors.New("tool registry is not available")
	}
	exec, ok := r.execs[name]
	if !ok || exec == nil {
		return nil, errors.New("tool executor not found")
	}
	return exec(args)
}

// NewQueryExecutor creates a project-scoped query tool executor.
func NewQueryExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}

		records := []*models.Record{}
		provider := search.NewProvider(search.NewSimpleFieldResolver("id", "created", "updated", "project"))

		if rawFilter, ok := args["filter"]; ok && rawFilter != nil {
			provider.AddFilter(search.FilterData(cast.ToString(rawFilter)))
		}
		if rawSort, ok := args["sort"]; ok && rawSort != nil {
			provider.Sort(search.ParseSortFromString(cast.ToString(rawSort)))
		}
		if rawPage, ok := args["page"]; ok && rawPage != nil {
			provider.Page(cast.ToInt(rawPage))
		}
		if rawPerPage, ok := args["perPage"]; ok && rawPerPage != nil {
			provider.PerPage(cast.ToInt(rawPerPage))
		}
		if rawSkip, ok := args["skipTotal"]; ok && rawSkip != nil {
			provider.SkipTotal(cast.ToBool(rawSkip))
		}

		result, err := provider.Query(app.Dao().RecordQuery(collection)).Exec(&records)
		if err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "query executed",
			Data:    result,
			Chart:   recommendChart(collection),
		}, nil
	}
}

// recommendChart inspects a collection schema and proposes a default
// visualization for query results (proposal §10.1). It never guarantees a
// chart is meaningful — it only provides a sensible default the UI can switch.
func recommendChart(collection *models.Collection) *ChartHint {
	if collection == nil {
		return &ChartHint{Type: "table"}
	}

	var dateField, categoryField string
	var numericFields []string

	for _, f := range collection.Schema.Fields() {
		switch f.Type {
		case schema.FieldTypeNumber:
			numericFields = append(numericFields, f.Name)
		case schema.FieldTypeDate:
			if dateField == "" {
				dateField = f.Name
			}
		case schema.FieldTypeSelect, schema.FieldTypeText:
			if categoryField == "" {
				categoryField = f.Name
			}
		}
	}

	if len(numericFields) == 0 {
		return &ChartHint{Type: "table"}
	}

	// time series => line chart
	if dateField != "" {
		return &ChartHint{Type: "line", XField: dateField, YFields: numericFields}
	}
	// categorical => bar chart
	if categoryField != "" {
		return &ChartHint{Type: "bar", XField: categoryField, YFields: numericFields}
	}
	// single numeric column => metric card
	if len(numericFields) == 1 {
		return &ChartHint{Type: "metric", YFields: numericFields}
	}
	return &ChartHint{Type: "table"}
}

// NewCreateTableExecutor creates a project-scoped collection creation executor.
func NewCreateTableExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		name := cast.ToString(args["name"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if name == "" {
			return nil, errors.New("name is required")
		}

		if !app.Dao().IsCollectionNameUnique(name) {
			return nil, errors.New("collection name must be unique")
		}

		fields, _ := args["fields"].([]any)
		schemaFields := make([]*schema.SchemaField, 0, len(fields))
		for _, raw := range fields {
			fieldMap, _ := raw.(map[string]any)
			if fieldMap == nil {
				continue
			}

			field := &schema.SchemaField{
				Name:     cast.ToString(fieldMap["name"]),
				Type:     cast.ToString(fieldMap["type"]),
				Required: cast.ToBool(fieldMap["required"]),
				Remark:   cast.ToString(fieldMap["remark"]),
			}
			if id := cast.ToString(fieldMap["id"]); id != "" {
				field.Id = id
			}
			if options := fieldMap["options"]; options != nil {
				field.Options = options
			}
			if err := field.InitOptions(); err != nil {
				return nil, err
			}
			schemaFields = append(schemaFields, field)
		}

		collection := &models.Collection{
			Name:    name,
			Type:    models.CollectionTypeBase,
			Project: &project,
		}
		if displayName := cast.ToString(args["displayName"]); displayName != "" {
			collection.DisplayName = &displayName
		} else {
			collection.DisplayName = &name
		}
		collection.Schema = schema.NewSchema(schemaFields...)
		if len(schemaFields) == 0 {
			return nil, errors.New("at least one field is required")
		}

		form := forms.NewCollectionUpsert(app, collection)
		form.Name = name
		form.Project = &project
		form.Type = models.CollectionTypeBase
		form.Schema = collection.Schema
		form.DisplayName = collection.DisplayName
		form.Options = types.JsonMap{}

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "table created",
			Data:    collection,
		}, nil
	}
}

// NewAddFieldExecutor updates a collection schema with a single new field.
func NewAddFieldExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		fieldMap, _ := args["field"].(map[string]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if fieldMap == nil {
			return nil, errors.New("field is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}
		if collection.IsView() {
			return nil, errors.New("view collections cannot be modified")
		}

		newField := &schema.SchemaField{
			Name:     cast.ToString(fieldMap["name"]),
			Type:     cast.ToString(fieldMap["type"]),
			Required: cast.ToBool(fieldMap["required"]),
			Remark:   cast.ToString(fieldMap["remark"]),
		}
		if id := cast.ToString(fieldMap["id"]); id != "" {
			newField.Id = id
		}
		if options := fieldMap["options"]; options != nil {
			newField.Options = options
		}
		if err := newField.InitOptions(); err != nil {
			return nil, err
		}
		if newField.Name == "" || newField.Type == "" {
			return nil, errors.New("field name and type are required")
		}

		next := *collection
		next.MarkAsNotNew()
		if cloned, err := collection.Schema.Clone(); err == nil && cloned != nil {
			next.Schema = *cloned
		} else {
			next.Schema = schema.NewSchema()
		}
		next.Schema.AddField(newField)

		form := forms.NewCollectionUpsert(app, &next)
		form.Name = next.Name
		form.Project = next.Project
		form.Type = next.Type
		form.DisplayName = next.DisplayName
		form.Schema = next.Schema
		form.Options = next.Options
		form.Indexes = next.Indexes
		form.ListRule = next.ListRule
		form.ViewRule = next.ViewRule
		form.CreateRule = next.CreateRule
		form.UpdateRule = next.UpdateRule
		form.DeleteRule = next.DeleteRule
		form.CacheEnabled = next.CacheEnabled
		form.ListCacheEnabled = next.ListCacheEnabled
		form.SearchCacheEnabled = next.SearchCacheEnabled
		form.CacheDuration = next.CacheDuration

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "field added",
			Data:    next.Schema,
		}, nil
	}
}

// NewDatasetPreviewExecutor returns a light-weight preview over the project's records.
func NewDatasetPreviewExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}

		records := []*models.Record{}
		err = app.Dao().RecordQuery(collection).
			OrderBy("updated DESC").
			Limit(10).
			All(&records)
		if err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "dataset preview generated",
			Data: map[string]any{
				"records": records,
			},
		}, nil
	}
}

// NewInsertRecordExecutor creates a project-scoped record insert executor.
func NewInsertRecordExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		data, _ := args["data"].(map[string]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if data == nil {
			return nil, errors.New("data is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}

		record := models.NewRecord(collection)
		form := forms.NewRecordUpsert(app, record)
		if err := form.LoadData(data); err != nil {
			return nil, err
		}
		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "record inserted",
			Data:    record,
		}, nil
	}
}

// NewBulkInsertRecordExecutor creates a project-scoped record bulk insert executor.
func NewBulkInsertRecordExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		rows, _ := args["rows"].([]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if len(rows) == 0 {
			return nil, errors.New("rows is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}

		inserted := []*models.Record{}
		err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			txCollection, err := txDao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return err
			}
			for _, raw := range rows {
				data, _ := raw.(map[string]any)
				if data == nil {
					return errors.New("each row must be an object")
				}

				record := models.NewRecord(txCollection)
				form := forms.NewRecordUpsert(app, record)
				form.SetDao(txDao)
				if err := form.LoadData(data); err != nil {
					return err
				}
				if err := form.Submit(); err != nil {
					return err
				}
				inserted = append(inserted, record)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "records inserted",
			Data: map[string]any{
				"records": inserted,
			},
		}, nil
	}
}

// NewGetRecordExecutor fetches a single project-scoped record.
func NewGetRecordExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		recordID := cast.ToString(args["id"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if recordID == "" {
			return nil, errors.New("id is required")
		}

		record, err := app.Dao().FindRecordById(collectionName, recordID)
		if err != nil {
			return nil, err
		}
		if record.Collection().Project == nil || *record.Collection().Project != project {
			return nil, errors.New("record is outside of the current project scope")
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "record fetched",
			Data:    record,
		}, nil
	}
}

// NewUpdateRecordExecutor creates a project-scoped record update executor.
func NewUpdateRecordExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		recordID := cast.ToString(args["id"])
		data, _ := args["data"].(map[string]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if recordID == "" {
			return nil, errors.New("id is required")
		}
		if data == nil {
			return nil, errors.New("data is required")
		}

		record, err := app.Dao().FindRecordById(collectionName, recordID)
		if err != nil {
			return nil, err
		}
		if record.Collection().Project == nil || *record.Collection().Project != project {
			return nil, errors.New("record is outside of the current project scope")
		}

		form := forms.NewRecordUpsert(app, record)
		if err := form.LoadData(data); err != nil {
			return nil, err
		}
		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "record updated",
			Data:    record,
		}, nil
	}
}

// NewDeleteRecordExecutor creates a project-scoped record delete executor.
func NewDeleteRecordExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		recordID := cast.ToString(args["id"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if recordID == "" {
			return nil, errors.New("id is required")
		}

		record, err := app.Dao().FindRecordById(collectionName, recordID)
		if err != nil {
			return nil, err
		}
		if record.Collection().Project == nil || *record.Collection().Project != project {
			return nil, errors.New("record is outside of the current project scope")
		}

		if err := app.Dao().Delete(record); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "record deleted",
			Data: map[string]any{
				"id": recordID,
			},
		}, nil
	}
}

// NewUpdateFieldExecutor updates an existing field definition.
func NewUpdateFieldExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		fieldMap, _ := args["field"].(map[string]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if fieldMap == nil {
			return nil, errors.New("field is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}
		if collection.IsView() {
			return nil, errors.New("view collections cannot be modified")
		}

		target := collection.Schema.GetFieldById(cast.ToString(fieldMap["id"]))
		if target == nil {
			target = collection.Schema.GetFieldByName(cast.ToString(fieldMap["name"]))
		}
		if target == nil {
			return nil, errors.New("field not found")
		}

		nextField := &schema.SchemaField{
			Id:       target.Id,
			Name:     target.Name,
			Type:     target.Type,
			Required: target.Required,
			Remark:   target.Remark,
			Options:  target.Options,
		}
		if v := cast.ToString(fieldMap["name"]); v != "" {
			nextField.Name = v
		}
		if v := cast.ToString(fieldMap["type"]); v != "" {
			nextField.Type = v
		}
		if nextField.Type != target.Type {
			return nil, errors.New("field type changes are not supported by this tool")
		}
		if v, ok := fieldMap["required"]; ok {
			nextField.Required = cast.ToBool(v)
		}
		if v := cast.ToString(fieldMap["remark"]); v != "" {
			nextField.Remark = v
		}
		if v, ok := fieldMap["options"]; ok && v != nil {
			nextField.Options = v
		}
		if err := nextField.InitOptions(); err != nil {
			return nil, err
		}

		next := *collection
		next.MarkAsNotNew()
		if cloned, err := collection.Schema.Clone(); err == nil && cloned != nil {
			next.Schema = *cloned
		} else {
			next.Schema = schema.NewSchema()
		}
		next.Schema.RemoveField(target.Id)
		next.Schema.AddField(nextField)

		form := forms.NewCollectionUpsert(app, &next)
		form.Name = next.Name
		form.Project = next.Project
		form.Type = next.Type
		form.DisplayName = next.DisplayName
		form.Schema = next.Schema
		form.Options = next.Options
		form.Indexes = next.Indexes
		form.ListRule = next.ListRule
		form.ViewRule = next.ViewRule
		form.CreateRule = next.CreateRule
		form.UpdateRule = next.UpdateRule
		form.DeleteRule = next.DeleteRule
		form.CacheEnabled = next.CacheEnabled
		form.ListCacheEnabled = next.ListCacheEnabled
		form.SearchCacheEnabled = next.SearchCacheEnabled
		form.CacheDuration = next.CacheDuration

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "field updated",
			Data:    next.Schema,
		}, nil
	}
}

// NewDropFieldExecutor removes a field from an existing table definition.
func NewDropFieldExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		fieldMap, _ := args["field"].(map[string]any)
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if fieldMap == nil {
			return nil, errors.New("field is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}
		if collection.IsView() {
			return nil, errors.New("view collections cannot be modified")
		}

		target := collection.Schema.GetFieldById(cast.ToString(fieldMap["id"]))
		if target == nil {
			target = collection.Schema.GetFieldByName(cast.ToString(fieldMap["name"]))
		}
		if target == nil {
			return nil, errors.New("field not found")
		}

		next := *collection
		next.MarkAsNotNew()
		if cloned, err := collection.Schema.Clone(); err == nil && cloned != nil {
			next.Schema = *cloned
		} else {
			next.Schema = schema.NewSchema()
		}
		next.Schema.RemoveField(target.Id)
		if len(next.Schema.Fields()) == 0 {
			return nil, errors.New("at least one field is required")
		}

		form := forms.NewCollectionUpsert(app, &next)
		form.Name = next.Name
		form.Project = next.Project
		form.Type = next.Type
		form.DisplayName = next.DisplayName
		form.Schema = next.Schema
		form.Options = next.Options
		form.Indexes = next.Indexes
		form.ListRule = next.ListRule
		form.ViewRule = next.ViewRule
		form.CreateRule = next.CreateRule
		form.UpdateRule = next.UpdateRule
		form.DeleteRule = next.DeleteRule
		form.CacheEnabled = next.CacheEnabled
		form.ListCacheEnabled = next.ListCacheEnabled
		form.SearchCacheEnabled = next.SearchCacheEnabled
		form.CacheDuration = next.CacheDuration

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "field dropped",
			Data:    next.Schema,
		}, nil
	}
}

// NewCreateIndexExecutor adds or replaces a collection index definition.
func NewCreateIndexExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		rawIndex := cast.ToString(args["index"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if rawIndex == "" {
			return nil, errors.New("index is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}
		if collection.IsView() {
			return nil, errors.New("view collections cannot be modified")
		}

		parsed := dbutils.ParseIndex(rawIndex)
		if !parsed.IsValid() {
			return nil, errors.New("invalid CREATE INDEX expression")
		}
		parsed.TableName = collection.Name

		next := *collection
		next.MarkAsNotNew()
		next.Indexes = append(types.JsonArray[string](nil), collection.Indexes...)
		normalized := parsed.Build()
		if normalized == "" {
			return nil, errors.New("invalid CREATE INDEX expression")
		}

		replaced := false
		for i, idx := range next.Indexes {
			current := dbutils.ParseIndex(idx)
			if current.IndexName != "" && strings.EqualFold(current.IndexName, parsed.IndexName) {
				next.Indexes[i] = normalized
				replaced = true
				break
			}
		}
		if !replaced {
			next.Indexes = append(next.Indexes, normalized)
		}

		form := forms.NewCollectionUpsert(app, &next)
		form.Name = next.Name
		form.Project = next.Project
		form.Type = next.Type
		form.DisplayName = next.DisplayName
		form.Schema = next.Schema
		form.Options = next.Options
		form.Indexes = next.Indexes
		form.ListRule = next.ListRule
		form.ViewRule = next.ViewRule
		form.CreateRule = next.CreateRule
		form.UpdateRule = next.UpdateRule
		form.DeleteRule = next.DeleteRule
		form.CacheEnabled = next.CacheEnabled
		form.ListCacheEnabled = next.ListCacheEnabled
		form.SearchCacheEnabled = next.SearchCacheEnabled
		form.CacheDuration = next.CacheDuration

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "index created",
			Data: map[string]any{
				"indexes": next.Indexes,
			},
		}, nil
	}
}

// NewSetRelationExecutor creates or updates a relation field definition.
func NewSetRelationExecutor(app core.App) ToolExecutor {
	return func(args map[string]any) (*ToolExecutionResult, error) {
		project := cast.ToString(args["project"])
		collectionName := cast.ToString(args["collection"])
		fieldMap, _ := args["field"].(map[string]any)
		relationRef := cast.ToString(args["relation"])
		if project == "" {
			return nil, errors.New("project is required")
		}
		if collectionName == "" {
			return nil, errors.New("collection is required")
		}
		if fieldMap == nil {
			return nil, errors.New("field is required")
		}
		if relationRef == "" {
			return nil, errors.New("relation is required")
		}

		collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
		if err != nil {
			return nil, err
		}
		if collection.Project == nil || *collection.Project != project {
			return nil, errors.New("collection is outside of the current project scope")
		}
		if collection.IsView() {
			return nil, errors.New("view collections cannot be modified")
		}

		relCollection, err := app.Dao().FindCollectionByNameOrId(relationRef)
		if err != nil {
			return nil, err
		}
		if relCollection == nil {
			return nil, errors.New("relation collection not found")
		}
		if relCollection.Project == nil || *relCollection.Project != project {
			return nil, errors.New("relation collection is outside of the current project scope")
		}

		fieldID := cast.ToString(fieldMap["id"])
		fieldName := cast.ToString(fieldMap["name"])
		var target *schema.SchemaField
		if fieldID != "" {
			target = collection.Schema.GetFieldById(fieldID)
		}
		if target == nil && fieldName != "" {
			target = collection.Schema.GetFieldByName(fieldName)
		}
		if target != nil && target.Type != schema.FieldTypeRelation {
			return nil, errors.New("field type changes are not supported by this tool")
		}

		maxSelect := func() *int {
			if v, ok := args["maxSelect"]; ok && v != nil {
				n := cast.ToInt(v)
				return &n
			}
			return nil
		}()
		minSelect := func() *int {
			if v, ok := args["minSelect"]; ok && v != nil {
				n := cast.ToInt(v)
				return &n
			}
			return nil
		}()

		displayFields := []string{}
		if raw, ok := args["displayFields"]; ok && raw != nil {
			if arr, ok := raw.([]any); ok {
				displayFields = make([]string, 0, len(arr))
				for _, item := range arr {
					if s := cast.ToString(item); s != "" {
						displayFields = append(displayFields, s)
					}
				}
			}
		}

		nextField := &schema.SchemaField{
			System: false,
			Id:     fieldID,
			Name:   fieldName,
			Type:   schema.FieldTypeRelation,
			Options: &schema.RelationOptions{
				CollectionId:  relCollection.Id,
				CascadeDelete: cast.ToBool(args["cascadeDelete"]),
				MinSelect:     minSelect,
				MaxSelect:     maxSelect,
				DisplayFields: displayFields,
			},
		}
		if nextField.Name == "" {
			return nil, errors.New("field name is required")
		}
		if nextField.Id == "" {
			nextField.Id = security.PseudorandomString(8)
		}
		if target != nil {
			nextField.Id = target.Id
			if nextField.Name == "" {
				nextField.Name = target.Name
			}
			if v, ok := fieldMap["required"]; ok {
				nextField.Required = cast.ToBool(v)
			} else {
				nextField.Required = target.Required
			}
			if v, ok := fieldMap["remark"]; ok && cast.ToString(v) != "" {
				nextField.Remark = cast.ToString(v)
			} else {
				nextField.Remark = target.Remark
			}
		} else {
			nextField.Required = cast.ToBool(fieldMap["required"])
			nextField.Remark = cast.ToString(fieldMap["remark"])
		}
		if err := nextField.InitOptions(); err != nil {
			return nil, err
		}

		next := *collection
		next.MarkAsNotNew()
		if cloned, err := collection.Schema.Clone(); err == nil && cloned != nil {
			next.Schema = *cloned
		} else {
			next.Schema = schema.NewSchema()
		}
		if target != nil {
			next.Schema.RemoveField(target.Id)
		}
		next.Schema.AddField(nextField)

		form := forms.NewCollectionUpsert(app, &next)
		form.Name = next.Name
		form.Project = next.Project
		form.Type = next.Type
		form.DisplayName = next.DisplayName
		form.Schema = next.Schema
		form.Options = next.Options
		form.Indexes = next.Indexes
		form.ListRule = next.ListRule
		form.ViewRule = next.ViewRule
		form.CreateRule = next.CreateRule
		form.UpdateRule = next.UpdateRule
		form.DeleteRule = next.DeleteRule
		form.CacheEnabled = next.CacheEnabled
		form.ListCacheEnabled = next.ListCacheEnabled
		form.SearchCacheEnabled = next.SearchCacheEnabled
		form.CacheDuration = next.CacheDuration

		if err := form.Submit(); err != nil {
			return nil, err
		}

		return &ToolExecutionResult{
			Status:  "ok",
			Message: "relation field updated",
			Data:    nextField,
		}, nil
	}
}
