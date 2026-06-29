package apis

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/spf13/cast"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/zhenruyan/postgrebase/vector"
)

// bindVectorApi registers embedded vector runtime endpoints.
func bindVectorApi(app core.App, rg *echo.Group) {
	api := vectorApi{app: app}

	subGroup := rg.Group("/vector", ActivityLogger(app))

	// admin-facing monitoring endpoints
	subGroup.GET("/status", api.status, RequireAdminAuth())
	subGroup.GET("/metrics", api.metrics, RequireAdminAuth())
	subGroup.GET("/cluster", api.cluster, RequireAdminAuth())
	subGroup.POST("/rebuild/:collection", api.rebuildCollection, RequireAdminAuth())

	// internal cluster transport endpoints (peer-to-peer)
	subGroup.POST("/cluster/heartbeat", api.heartbeat)
	subGroup.POST("/cluster/replicate", api.replicate)
	subGroup.POST("/cluster/forward", api.forward)
	subGroup.POST("/cluster/install-snapshot", api.installSnapshot)
	subGroup.POST("/cluster/join", api.join)
}

type vectorApi struct {
	app core.App
}

func (api *vectorApi) status(c echo.Context) error {
	manager := api.app.VectorManager()
	if manager == nil {
		return c.JSON(http.StatusOK, map[string]any{"enabled": false})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"runtime": manager.Status(),
	})
}

func (api *vectorApi) metrics(c echo.Context) error {
	manager := api.app.VectorManager()
	if manager == nil {
		return c.JSON(http.StatusOK, map[string]any{"enabled": false})
	}
	metrics := manager.Metrics()
	metrics.CacheItems = api.cacheItems(c.Request().Context())
	metrics.CacheBackend = api.cacheBackend()
	metrics.RedisEnabled = metrics.CacheBackend == "redis"
	return c.JSON(http.StatusOK, map[string]any{
		"metrics": metrics,
	})
}

func (api *vectorApi) cacheBackend() string {
	if api.app.RedisCache() != nil {
		return "redis"
	}
	return "memory"
}

func (api *vectorApi) cacheItems(ctx context.Context) int {
	if api.app.RedisCache() == nil {
		return api.app.Cache().LengthByPrefix("pb_cache:")
	}

	iter := api.app.RedisCache().Scan(ctx, 0, "pb_cache:*", 0).Iterator()
	count := 0
	for iter.Next(ctx) {
		count++
	}
	return count
}

func (api *vectorApi) cluster(c echo.Context) error {
	manager := api.app.VectorManager()
	if manager == nil {
		return c.JSON(http.StatusOK, map[string]any{"enabled": false})
	}

	coordinator := manager.Coordinator()
	if coordinator == nil {
		// standalone mode without coordinator
		status := manager.Status()
		return c.JSON(http.StatusOK, map[string]any{
			"mode": vector.ModeStandalone,
			"view": vector.ClusterView{
				Mode:     vector.ModeStandalone,
				NodeID:   status.NodeID,
				IsLeader: true,
			},
		})
	}

	view := coordinator.View()
	return c.JSON(http.StatusOK, map[string]any{
		"mode": view.Mode,
		"view": view,
	})
}

func (api *vectorApi) coordinatorOrError() (*vector.Coordinator, error) {
	manager := api.app.VectorManager()
	if manager == nil {
		return nil, NewApiError(http.StatusServiceUnavailable, "Vector runtime is not enabled.", nil)
	}
	coordinator := manager.Coordinator()
	if coordinator == nil {
		return nil, NewApiError(http.StatusServiceUnavailable, "Cluster coordinator is not enabled.", nil)
	}
	return coordinator, nil
}

func (api *vectorApi) heartbeat(c echo.Context) error {
	coordinator, err := api.coordinatorOrError()
	if err != nil {
		return err
	}

	hb := vector.Heartbeat{}
	if err := c.Bind(&hb); err != nil {
		return NewBadRequestError("Failed to read heartbeat payload.", err)
	}

	return c.JSON(http.StatusOK, coordinator.ReceiveHeartbeat(hb))
}

func (api *vectorApi) replicate(c echo.Context) error {
	coordinator, err := api.coordinatorOrError()
	if err != nil {
		return err
	}

	op := vector.ReplicatedOperation{}
	if err := c.Bind(&op); err != nil {
		return NewBadRequestError("Failed to read replication payload.", err)
	}

	if _, err := coordinator.ApplyReplicated(op); err != nil {
		return NewApiError(http.StatusInternalServerError, "Failed to apply replicated operation.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"applied": true})
}

func (api *vectorApi) forward(c echo.Context) error {
	coordinator, err := api.coordinatorOrError()
	if err != nil {
		return err
	}

	op := vector.ReplicatedOperation{}
	if err := c.Bind(&op); err != nil {
		return NewBadRequestError("Failed to read forwarded payload.", err)
	}

	if _, err := coordinator.ProposeReplicated(op); err != nil {
		return NewApiError(http.StatusInternalServerError, "Failed to apply forwarded operation.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"applied": true})
}

func (api *vectorApi) rebuildCollection(c echo.Context) error {
	collectionIdOrName := c.PathParam("collection")
	collection, err := api.app.Dao().FindCollectionByNameOrId(collectionIdOrName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "Collection not found"})
	}

	mgr := api.app.VectorManager()
	if mgr == nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Vector manager not enabled"})
	}

	vectorDb := api.app.VectorDB()
	if vectorDb == nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Vector database not available"})
	}

	// 1. Delete existing entries and tasks for this collection in the local SQLite vector database!
	sourceType := "record:" + collection.Id
	_, _ = vectorDb.NewQuery("DELETE FROM _pb_vector_tasks_ WHERE source_type = {:sourceType}").
		Bind(dbx.Params{"sourceType": sourceType}).
		Execute()
	_, _ = vectorDb.NewQuery("DELETE FROM _pb_vector_entries_ WHERE source_type = {:sourceType}").
		Bind(dbx.Params{"sourceType": sourceType}).
		Execute()

	// Reload to sync in-memory cache
	_ = mgr.Load()

	// 2. Fetch all records from the collection
	records, err := api.app.Dao().FindRecordsByExpr(collection.Id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	count := 0
	for _, record := range records {
		// Clear vector field value in main record table
		for _, f := range collection.Schema.Fields() {
			if f.Type == "vector" {
				record.Set(f.Name, "")
			}
		}
		_ = api.app.Dao().SaveRecord(record)

		queuedIds := mgr.TriggerRecordEmbedding(record)
		count += len(queuedIds)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"tasks":   count,
	})
}

func (api *vectorApi) installSnapshot(c echo.Context) error {
	coordinator, err := api.coordinatorOrError()
	if err != nil {
		return err
	}

	lastLogIndex := cast.ToUint64(c.QueryParam("lastLogIndex"))
	appliedLogIndex := cast.ToUint64(c.QueryParam("appliedLogIndex"))
	term := cast.ToUint64(c.QueryParam("term"))

	// Create a temp file path to write incoming snapshot
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("pb_incoming_snapshot_%d.db", time.Now().UnixNano()))
	defer os.Remove(tempPath)

	file, err := os.Create(tempPath)
	if err != nil {
		return NewBadRequestError("Failed to create temporary snapshot file.", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, c.Request().Body); err != nil {
		return NewBadRequestError("Failed to stream incoming snapshot payload.", err)
	}
	file.Close()

	// Install snapshot into coordinator
	if err := coordinator.InstallSnapshot(tempPath, lastLogIndex, appliedLogIndex, term); err != nil {
		return NewApiError(http.StatusInternalServerError, "Failed to install snapshot.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"installed": true})
}

func (api *vectorApi) join(c echo.Context) error {
	coordinator, err := api.coordinatorOrError()
	if err != nil {
		return err
	}

	var req struct {
		Addr string `json:"addr"`
	}
	if err := c.Bind(&req); err != nil {
		return NewBadRequestError("Failed to read join payload.", err)
	}

	if req.Addr == "" {
		return NewBadRequestError("Addr is required.", nil)
	}

	op := vector.ReplicatedOperation{
		ID:        security.NewUUIDString(),
		Kind:      vector.ReplicatedOperationKindSQLite,
		Type:      "cluster.join",
		Strict:    true,
		Payload:   []byte(fmt.Sprintf(`{"addr":%q}`, req.Addr)),
		CreatedAt: time.Now().UTC(),
	}

	if _, err := coordinator.ProposeReplicated(op); err != nil {
		return NewApiError(http.StatusInternalServerError, "Failed to join cluster.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"joined": true})
}
