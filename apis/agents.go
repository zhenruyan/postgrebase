package apis

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/agents"
	"github.com/zhenruyan/postgrebase/core"
)

func bindAgentsApi(app core.App, rg *echo.Group) {
	api := agentsApi{svc: agents.NewService(app)}

	subGroup := rg.Group("/agents", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("", api.runtime)
	subGroup.GET("/providers", api.providers)
	subGroup.GET("/models", api.models)
}

type agentsApi struct {
	svc *agents.Service
}

func (api *agentsApi) runtime(c echo.Context) error {
	return c.JSON(http.StatusOK, api.svc.Runtime())
}

func (api *agentsApi) providers(c echo.Context) error {
	return c.JSON(http.StatusOK, api.svc.Providers())
}

func (api *agentsApi) models(c echo.Context) error {
	return c.JSON(http.StatusOK, api.svc.Models())
}
