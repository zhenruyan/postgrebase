package apis

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
)

// bindVectorApi registers embedded vector runtime endpoints.
func bindVectorApi(app core.App, rg *echo.Group) {
	api := vectorApi{app: app}

	subGroup := rg.Group("/vector", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("/status", api.status)
}

type vectorApi struct {
	app core.App
}

func (api *vectorApi) status(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"runtime": api.app.VectorManager().Status(),
	})
}
