package agents

import (
	"testing"

	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
)

func collectionWith(fields ...*schema.SchemaField) *models.Collection {
	c := &models.Collection{}
	c.Schema = schema.NewSchema(fields...)
	return c
}

func TestRecommendChart(t *testing.T) {
	// time series => line
	line := recommendChart(collectionWith(
		&schema.SchemaField{Name: "day", Type: schema.FieldTypeDate},
		&schema.SchemaField{Name: "amount", Type: schema.FieldTypeNumber},
	))
	if line.Type != "line" || line.XField != "day" || len(line.YFields) != 1 || line.YFields[0] != "amount" {
		t.Fatalf("expected line chart on date+number, got %+v", line)
	}

	// categorical => bar
	bar := recommendChart(collectionWith(
		&schema.SchemaField{Name: "category", Type: schema.FieldTypeSelect},
		&schema.SchemaField{Name: "total", Type: schema.FieldTypeNumber},
	))
	if bar.Type != "bar" || bar.XField != "category" {
		t.Fatalf("expected bar chart on select+number, got %+v", bar)
	}

	// single numeric => metric
	metric := recommendChart(collectionWith(
		&schema.SchemaField{Name: "count", Type: schema.FieldTypeNumber},
	))
	if metric.Type != "metric" {
		t.Fatalf("expected metric on single number, got %+v", metric)
	}

	// no numeric => table
	table := recommendChart(collectionWith(
		&schema.SchemaField{Name: "title", Type: schema.FieldTypeText},
	))
	if table.Type != "table" {
		t.Fatalf("expected table with no numeric field, got %+v", table)
	}

	// nil collection => table
	if recommendChart(nil).Type != "table" {
		t.Fatal("expected table for nil collection")
	}
}
