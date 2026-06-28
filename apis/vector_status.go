package apis

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
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

	// internal cluster transport endpoints (peer-to-peer)
	subGroup.POST("/cluster/heartbeat", api.heartbeat)
	subGroup.POST("/cluster/replicate", api.replicate)
	subGroup.POST("/cluster/forward", api.forward)
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
	return c.JSON(http.StatusOK, map[string]any{
		"metrics": manager.Metrics(),
	})
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

	op := vector.Operation{}
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

	op := vector.Operation{}
	if err := c.Bind(&op); err != nil {
		return NewBadRequestError("Failed to read forwarded payload.", err)
	}

	if _, err := coordinator.Propose(op); err != nil {
		return NewApiError(http.StatusInternalServerError, "Failed to apply forwarded operation.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"applied": true})
}
