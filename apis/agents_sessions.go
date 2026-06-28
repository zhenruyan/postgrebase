package apis

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/agents"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/models"
)

func bindAgentSessionApi(app core.App, rg *echo.Group) {
	api := agentSessionApi{svc: agents.NewService(app)}

	subGroup := rg.Group("/agents", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("/sessions", api.list)
	subGroup.GET("/sessions/:id", api.view)
	subGroup.GET("/sessions/:id/audit", api.audit)
	subGroup.POST("/sessions", api.create)
	subGroup.PATCH("/sessions/:id", api.rename)
	subGroup.POST("/sessions/:id/messages", api.message)
	subGroup.POST("/sessions/:id/run", api.run)
	subGroup.GET("/tools", api.tools)
	subGroup.POST("/tools/:name", api.callTool)
}

type agentSessionApi struct {
	svc *agents.Service
}

func (api *agentSessionApi) list(c echo.Context) error {
	project := c.QueryParam("project")
	return c.JSON(http.StatusOK, api.svc.ListSessions(project))
}

func (api *agentSessionApi) create(c echo.Context) error {
	var body struct {
		Project  string `json:"project"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}
	if body.Project == "" {
		return NewBadRequestError("Project is required", nil)
	}
	session := api.svc.CreateSession(body.Project, body.Name, body.Provider, body.Model)
	return c.JSON(http.StatusOK, session)
}

func (api *agentSessionApi) rename(c echo.Context) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Session ID is required", nil)
	}

	session, err := api.svc.RenameSession(id, body.Name)
	if err != nil {
		return NewBadRequestError("Failed to rename session", err)
	}
	return c.JSON(http.StatusOK, session)
}

func (api *agentSessionApi) view(c echo.Context) error {
	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Session ID is required", nil)
	}

	session, err := api.svc.GetSession(id)
	if err != nil {
		return NewNotFoundError("", err)
	}

	messages, err := api.svc.SessionMessages(id)
	if err != nil {
		return NewNotFoundError("", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"session":  session,
		"messages": messages,
	})
}

func (api *agentSessionApi) message(c echo.Context) error {
	var body struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Session ID is required", nil)
	}

	session, messages, err := api.svc.AppendMessage(id, body.Role, body.Content)
	if err != nil {
		return NewNotFoundError("", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"session":  session,
		"messages": messages,
	})
}

func (api *agentSessionApi) run(c echo.Context) error {
	var body struct {
		Content       string                   `json:"content"`
		Images        []agents.AgentImageInput `json:"images"`
		AllowWrites   bool                     `json:"allowWrites"`
		ApprovedTools []string                 `json:"approvedTools"`
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Session ID is required", nil)
	}

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return NewBadRequestError("Streaming is not supported by this server", nil)
	}

	c.Response().Header().Set("Content-Type", "text/event-stream; charset=UTF-8")
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	opts := agents.RunOptions{
		AllowWrites:   body.AllowWrites,
		ApprovedTools: body.ApprovedTools,
		Actor:         actorFromContext(c),
	}
	input := agents.RunInput{Content: body.Content, Images: body.Images}

	var writeErr error
	var sentError bool
	_, err := api.svc.RunSessionStream(c.Request().Context(), id, input, opts, func(ev agents.RunStreamEvent) bool {
		if ev.Type == agents.RunStreamEventError {
			sentError = true
		}
		writeErr = writeAgentRunEvent(c, flusher, ev)
		return writeErr == nil
	})
	if err != nil {
		if writeErr != nil || c.Request().Context().Err() != nil {
			return nil
		}
		log.Printf("agents: run session %s failed: %v", id, err)
		if !sentError {
			_ = writeAgentRunEvent(c, flusher, agents.RunStreamEvent{
				Type:  agents.RunStreamEventError,
				Error: "Failed to run agent session: " + err.Error(),
			})
		}
	}

	return nil
}

func writeAgentRunEvent(c echo.Context, flusher http.Flusher, ev agents.RunStreamEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Response(), "event: %s\n", ev.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// actorFromContext extracts the authenticated admin id for audit purposes.
func actorFromContext(c echo.Context) string {
	if admin, _ := c.Get(ContextAdminKey).(*models.Admin); admin != nil {
		return "admin:" + admin.Id
	}
	return ""
}

func (api *agentSessionApi) audit(c echo.Context) error {
	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Session ID is required", nil)
	}
	records, err := api.svc.SessionAudit(id)
	if err != nil {
		return NewBadRequestError("Failed to load audit trail", err)
	}
	return c.JSON(http.StatusOK, records)
}

func (api *agentSessionApi) tools(c echo.Context) error {
	return c.JSON(http.StatusOK, api.svc.Tools())
}

func (api *agentSessionApi) callTool(c echo.Context) error {
	name := c.PathParam("name")
	sessionID := c.QueryParam("session_id")
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	result, err := api.svc.ExecuteToolInSession(sessionID, name, body)
	if err != nil {
		return NewBadRequestError("Failed to execute tool", err)
	}

	history, _ := api.svc.SessionMessages(sessionID)

	return c.JSON(http.StatusOK, map[string]any{
		"tool":    name,
		"result":  result,
		"session": sessionID,
		"history": history,
	})
}
