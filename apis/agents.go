package apis

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/agents"
	"github.com/zhenruyan/postgrebase/core"
)

func bindAgentsApi(app core.App, rg *echo.Group) {
	api := agentsApi{svc: agents.NewService(app)}

	app.OnSettingsAfterUpdateRequest().Add(func(e *core.SettingsUpdateEvent) error {
		api.svc.Refresh()
		return nil
	})

	subGroup := rg.Group("/agents", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("", api.runtime)
	subGroup.GET("/providers", api.providers)
	subGroup.GET("/models", api.models)
	subGroup.GET("/projects/:project/config", api.projectConfig)
	subGroup.PUT("/projects/:project/config", api.saveProjectConfig)
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

func (api *agentsApi) projectConfig(c echo.Context) error {
	project := c.PathParam("project")
	if project == "" {
		return NewBadRequestError("Project is required", nil)
	}
	return c.JSON(http.StatusOK, api.svc.GetProjectConfig(project))
}

func (api *agentsApi) saveProjectConfig(c echo.Context) error {
	project := c.PathParam("project")
	if project == "" {
		return NewBadRequestError("Project is required", nil)
	}

	var body agents.ProjectConfig
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}
	body.Project = project

	saved, err := api.svc.SaveProjectConfig(body)
	if err != nil {
		return NewBadRequestError("Failed to save project config: "+err.Error(), nil)
	}
	return c.JSON(http.StatusOK, saved)
}
