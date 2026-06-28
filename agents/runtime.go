package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	agentsdk "github.com/startvibecoding/vibecoding/agent"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/settings"
)

// maxRunIterations bounds the number of model<->tool round trips per run.
const maxRunIterations = 20

// RunTrace records a single tool execution performed during an agent run.
type RunTrace struct {
	Tool   string `json:"tool"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// AgentImageInput is an image supplied with a user turn (proposal §6.1).
// Provide either inline base64 Data, or a FileRef to resolve from the file
// subsystem (proposal §6.2).
type AgentImageInput struct {
	MimeType string        `json:"mimeType"`
	Data     string        `json:"data"`
	FileRef  *AgentFileRef `json:"fileRef,omitempty"`
}

// RunInput is the user turn fed to the agent: text plus optional images.
type RunInput struct {
	Content string            `json:"content"`
	Images  []AgentImageInput `json:"images,omitempty"`
}

// RunResult is the normalized outcome of an agent run.
type RunResult struct {
	SessionId        string            `json:"sessionId"`
	SessionName      string            `json:"sessionName,omitempty"`
	Reply            string            `json:"reply"`
	Provider         string            `json:"provider"`
	Model            string            `json:"model"`
	Traces           []RunTrace        `json:"traces,omitempty"`
	PendingApprovals []PendingApproval `json:"pendingApprovals,omitempty"`
	Audit            []AgentAuditEntry `json:"audit,omitempty"`
	Messages         []SessionMessage  `json:"messages"`
}

// resolveProvider returns the provider/model configuration to use for a run,
// honoring the session-level overrides and falling back to defaults.
func (s *Service) resolveProvider(sessionProvider, sessionModel string) (settings.AgentProviderConfig, string, error) {
	cfg := s.app.Settings().Agents
	if !cfg.Enabled {
		return settings.AgentProviderConfig{}, "", errors.New("agent runtime is disabled")
	}
	if len(cfg.Providers) == 0 {
		return settings.AgentProviderConfig{}, "", errors.New("no agent providers configured")
	}

	providerId := strings.TrimSpace(sessionProvider)
	if providerId == "" {
		if inferredProviderId := providerIDByModel(cfg.Providers, sessionModel); inferredProviderId != "" {
			providerId = inferredProviderId
		} else {
			providerId = cfg.DefaultProvider
		}
	}

	var provider settings.AgentProviderConfig
	found := false
	for _, p := range cfg.Providers {
		if p.Id == providerId {
			provider = p
			found = true
			break
		}
	}
	if !found {
		return settings.AgentProviderConfig{}, "", fmt.Errorf("agent provider %q not found", providerId)
	}
	if !provider.Enabled {
		return settings.AgentProviderConfig{}, "", fmt.Errorf("agent provider %q is disabled", provider.Id)
	}

	model := strings.TrimSpace(sessionModel)
	if id := resolveModelId(provider, model); id != "" {
		return provider, id, nil
	}
	if model == "" {
		model = provider.DefaultModel
	}
	if model == "" {
		model = firstProviderModelId(provider)
	}
	if model == "" {
		model = cfg.DefaultModel
	}
	if id := resolveModelId(provider, model); id != "" {
		model = id
	}
	if model == "" {
		return settings.AgentProviderConfig{}, "", errors.New("no model configured for agent run")
	}

	return provider, model, nil
}

func firstProviderModelId(provider settings.AgentProviderConfig) string {
	for _, m := range provider.Models {
		if !m.Enabled {
			continue
		}
		if id := strings.TrimSpace(m.ProviderModelId); id != "" {
			return id
		}
		if name := strings.TrimSpace(m.Name); name != "" {
			return name
		}
	}
	for _, m := range provider.Models {
		if id := strings.TrimSpace(m.ProviderModelId); id != "" {
			return id
		}
		if name := strings.TrimSpace(m.Name); name != "" {
			return name
		}
	}
	return ""
}

func providerIDByModel(providers []settings.AgentProviderConfig, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	for _, provider := range providers {
		for _, m := range provider.Models {
			if !m.Enabled {
				continue
			}
			if strings.TrimSpace(m.ProviderModelId) == model || strings.TrimSpace(m.Name) == model {
				return provider.Id
			}
		}
	}
	for _, provider := range providers {
		for _, m := range provider.Models {
			if strings.TrimSpace(m.ProviderModelId) == model || strings.TrimSpace(m.Name) == model {
				return provider.Id
			}
		}
	}
	return ""
}

// resolveModelId maps a configured model name to its provider model id.
func resolveModelId(provider settings.AgentProviderConfig, model string) string {
	for _, m := range provider.Models {
		if m.Name == model || m.ProviderModelId == model {
			if m.ProviderModelId != "" {
				return m.ProviderModelId
			}
			return m.Name
		}
	}
	return ""
}

func configuredProviderModelExists(provider settings.AgentProviderConfig, model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	if strings.TrimSpace(provider.DefaultModel) == model {
		return true
	}
	return resolveModelId(provider, model) != ""
}

func configuredProviderById(providers []settings.AgentProviderConfig, id string) (settings.AgentProviderConfig, bool) {
	id = strings.TrimSpace(id)
	for _, provider := range providers {
		if provider.Id == id {
			return provider, true
		}
	}
	return settings.AgentProviderConfig{}, false
}

func sessionProviderUsesDefault(rawProvider, providerId, policyProvider string) bool {
	rawProvider = strings.TrimSpace(rawProvider)
	providerId = strings.TrimSpace(providerId)
	policyProvider = strings.TrimSpace(policyProvider)
	return rawProvider == "" || (policyProvider != "" && providerId == policyProvider)
}

func (s *Service) effectiveRunSelection(session *Session, policy projectPolicy) (string, string) {
	cfg := s.app.Settings().Agents
	rawProvider := strings.TrimSpace(session.Provider)
	providerId := rawProvider
	policyProvider := strings.TrimSpace(policy.defaultProvider)
	if providerId == "" {
		providerId = policyProvider
	} else if policy.projectProvider && policyProvider != "" && providerId == strings.TrimSpace(cfg.DefaultProvider) && providerId != policyProvider {
		providerId = policyProvider
	}

	model := strings.TrimSpace(session.Model)
	policyModel := strings.TrimSpace(policy.defaultModel)
	if policyModel != "" {
		provider, hasProvider := configuredProviderById(cfg.Providers, firstNonEmpty(providerId, policyProvider))
		usesDefaultProvider := sessionProviderUsesDefault(rawProvider, providerId, policyProvider)
		switch {
		case model == "":
			if usesDefaultProvider {
				model = policyModel
			}
		case model == strings.TrimSpace(cfg.DefaultModel) && model != policyModel:
			model = policyModel
		case usesDefaultProvider && policy.projectModel && hasProvider && !configuredProviderModelExists(provider, model):
			model = policyModel
		}
	}

	return providerId, model
}

// toolName converts a dotted tool name to a provider-safe function name.
// The chat completions API requires names matching ^[a-zA-Z0-9_-]+$.
func toolName(name string) string {
	return strings.ReplaceAll(name, ".", "__")
}

// apiStyle maps a vendor to the vibecoding provider API style.
func apiStyle(provider settings.AgentProviderConfig) string {
	switch strings.ToLower(strings.TrimSpace(provider.Api)) {
	case "openai-chat", "openai-responses", "anthropic-messages", "google-gemini", "google-vertex":
		return strings.ToLower(strings.TrimSpace(provider.Api))
	}

	switch strings.ToLower(strings.TrimSpace(provider.Vendor)) {
	case "anthropic":
		return "anthropic-messages"
	case "google-gemini", "gemini":
		return "google-gemini"
	case "google-vertex":
		return "google-vertex"
	default:
		return "openai-chat"
	}
}

// systemPrompt builds the run system prompt that fixes the project boundary
// and the tool-only data access contract (proposal §4.2, §4.4, §7).
func systemPrompt(project string) string {
	var b strings.Builder
	b.WriteString("You are the embedded data agent for the PostgreBase project ")
	b.WriteString(project)
	b.WriteString(".\n")
	b.WriteString("Rules:\n")
	b.WriteString("- All structured data access MUST go through the provided tools. Never emit raw SQL.\n")
	b.WriteString("- You may only operate within project_id=")
	b.WriteString(project)
	b.WriteString(". The project argument is injected automatically; never target another project.\n")
	b.WriteString("- Before guessing table or collection names, call schema.list_tables and use a returned table name.\n")
	b.WriteString("- Use schema tools to create or modify tables, and data tools to insert, query, update or delete records.\n")
	b.WriteString("- If a tool returns status=pending_approval, stop calling write tools and ask the user to approve.\n")
	b.WriteString("- When you have enough information, answer the user directly and concisely.\n")
	return b.String()
}

// modelSupportsVision reports whether the resolved model accepts image input.
func modelSupportsVision(provider settings.AgentProviderConfig, modelId string) bool {
	for _, m := range provider.Models {
		if m.ProviderModelId == modelId || m.Name == modelId {
			return m.SupportsVision
		}
	}
	return false
}

// historyToMessages converts stored session messages to SDK messages.
// Intermediate tool traces are dropped; only user/assistant turns are replayed.
// User turns that carry images are replayed as multimodal content blocks.
func historyToMessages(history []SessionMessage) []agentsdk.Message {
	messages := make([]agentsdk.Message, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "user":
			if len(m.Images) == 0 {
				messages = append(messages, agentsdk.NewUserMessage(m.Content))
				continue
			}
			blocks := make([]agentsdk.ContentBlock, 0, len(m.Images)+1)
			if strings.TrimSpace(m.Content) != "" {
				blocks = append(blocks, agentsdk.ContentBlock{Type: "text", Text: m.Content})
			}
			for _, img := range m.Images {
				blocks = append(blocks, agentsdk.ContentBlock{
					Type:  "image",
					Image: &agentsdk.ImageContent{MimeType: img.MimeType, Data: img.Data},
				})
			}
			messages = append(messages, agentsdk.Message{Role: agentsdk.RoleUser, Contents: blocks})
		case "assistant":
			messages = append(messages, agentsdk.NewAssistantTextMessage(m.Content))
		}
	}
	return messages
}

// RunSession stores the user message, drives the vibecoding agent runtime and
// returns the final accumulated result.
func (s *Service) RunSession(ctx context.Context, sessionID string, input RunInput, opts RunOptions) (*RunResult, error) {
	return s.RunSessionStream(ctx, sessionID, input, opts, nil)
}

func emitRunStreamEvent(ctx context.Context, emit RunStreamHandler, ev RunStreamEvent) error {
	if emit == nil {
		return nil
	}
	if emit(ev) {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return errors.New("agent stream closed")
}

func eventToolName(ev agentsdk.Event) string {
	if ev.ToolName != "" {
		return ev.ToolName
	}
	if ev.ToolCall != nil {
		return ev.ToolCall.Name
	}
	return ""
}

func eventToolCallID(ev agentsdk.Event) string {
	if ev.ToolCallID != "" {
		return ev.ToolCallID
	}
	if ev.ToolCall != nil {
		return ev.ToolCall.ID
	}
	return ""
}

func encodedToolArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	return string(raw)
}

// RunSessionStream stores the user message, drives the vibecoding agent runtime
// (model + tool loop), streams progress events and persists the final assistant
// and tool messages.
func (s *Service) RunSessionStream(ctx context.Context, sessionID string, input RunInput, opts RunOptions, emit RunStreamHandler) (*RunResult, error) {
	if s == nil || s.sessions == nil || s.tools == nil {
		return nil, errors.New("agent runtime is not available")
	}

	session, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	// Resolve the effective per-project policy (proposal §9.1) and overlay it on
	// the session-level provider/model selection.
	policy := s.resolvePolicy(session.Project)
	sessionProvider, sessionModel := s.effectiveRunSelection(session, policy)
	if policy.autoApprove {
		opts.AllowWrites = true
	}

	provider, model, err := s.resolveProvider(sessionProvider, sessionModel)
	if err != nil {
		return nil, err
	}

	// Resolve file-referenced images via the file subsystem (proposal §6.2),
	// enforcing project scope, before any capability checks or persistence.
	resolvedImages, err := s.resolveImageInputs(session.Project, input.Images)
	if err != nil {
		return nil, err
	}
	input.Images = resolvedImages

	// Validate multimodal capability before persisting anything (proposal §6.1).
	if len(input.Images) > 0 && !modelSupportsVision(provider, model) {
		return nil, fmt.Errorf("model %q does not support image input", model)
	}

	// persist the incoming user message (with any image attachments)
	if strings.TrimSpace(input.Content) != "" || len(input.Images) > 0 {
		images := make([]SessionImage, 0, len(input.Images))
		for _, img := range input.Images {
			if strings.TrimSpace(img.Data) == "" {
				continue
			}
			mime := strings.TrimSpace(img.MimeType)
			if mime == "" {
				mime = "image/png"
			}
			images = append(images, SessionImage{MimeType: mime, Data: img.Data})
		}
		if _, _, err := s.sessions.AddMessageWithImages(sessionID, "user", input.Content, images); err != nil {
			return nil, err
		}
	}

	result := &RunResult{
		SessionId: sessionID,
		Provider:  provider.Id,
		Model:     model,
	}
	if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
		Type:      RunStreamEventStart,
		SessionId: sessionID,
		Provider:  provider.Id,
		Model:     model,
	}); err != nil {
		return nil, err
	}

	history, err := s.sessions.Messages(sessionID)
	if err != nil {
		return nil, err
	}

	audit := &auditSink{session: sessionID, project: session.Project, actor: opts.Actor}
	tools := s.externalTools(session.Project, policy, opts, audit)

	agent, err := agentsdk.NewBuilder().
		WithProviderByName(provider.Vendor, provider.BaseUrl, apiStyle(provider), resolveApiKey(provider.ApiKey)).
		WithModel(model).
		WithMode("agent").
		WithoutBuiltinTools().
		WithExternalTools(tools...).
		WithSystemPromptExtra(systemPrompt(session.Project)).
		WithMaxIterations(maxRunIterations).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	var reply strings.Builder
	toolArgs := map[string]string{}
	events := agent.RunWithMessages(ctx, historyToMessages(history))
	for ev := range events {
		switch ev.Type {
		case agentsdk.EventTextDelta:
			reply.WriteString(ev.TextDelta)
			if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
				Type: RunStreamEventTextDelta,
				Text: ev.TextDelta,
			}); err != nil {
				return nil, err
			}
		case agentsdk.EventThinkDelta:
			if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
				Type:    RunStreamEventThinkDelta,
				Thought: ev.ThinkDelta,
			}); err != nil {
				return nil, err
			}
		case agentsdk.EventToolCall:
			tool := eventToolName(ev)
			if id := eventToolCallID(ev); id != "" {
				toolArgs[id] = encodedToolArgs(ev.ToolArgs)
			}
			if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
				Type: RunStreamEventToolCall,
				Tool: fromToolName(tool),
				Args: ev.ToolArgs,
			}); err != nil {
				return nil, err
			}
		case agentsdk.EventToolExecutionStart:
			if id := eventToolCallID(ev); id != "" {
				toolArgs[id] = encodedToolArgs(ev.ToolArgs)
			}
		case agentsdk.EventStatus:
			if strings.TrimSpace(ev.StatusMessage) == "" {
				continue
			}
			if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
				Type:   RunStreamEventStatus,
				Status: ev.StatusMessage,
			}); err != nil {
				return nil, err
			}
		case agentsdk.EventToolResult:
			trace := RunTrace{Tool: fromToolName(eventToolName(ev)), Args: toolArgs[eventToolCallID(ev)], Result: ev.ToolResult}
			if ev.ToolError != nil {
				trace.Error = ev.ToolError.Error()
			}
			result.Traces = append(result.Traces, trace)
			_, _, _ = s.sessions.AddMessage(sessionID, "tool", trace.Tool+": "+ev.ToolResult)
			if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
				Type:             RunStreamEventToolResult,
				Trace:            &trace,
				PendingApprovals: audit.pendings,
			}); err != nil {
				return nil, err
			}
		case agentsdk.EventError:
			if ev.Error != nil {
				_ = emitRunStreamEvent(ctx, emit, RunStreamEvent{
					Type:  RunStreamEventError,
					Error: ev.Error.Error(),
				})
				return nil, ev.Error
			}
		}
	}

	result.PendingApprovals = audit.pendings
	result.Audit = audit.entries
	result.Reply = strings.TrimSpace(reply.String())
	if result.Reply == "" {
		result.Reply = fallbackRunReply(result.PendingApprovals, result.Traces)
	}
	s.persistAudit(sessionID, session.Project, audit.entries)
	if result.Reply != "" {
		if _, _, err := s.sessions.AddMessage(sessionID, "assistant", result.Reply); err != nil {
			return nil, err
		}
	}

	if msgs, mErr := s.sessions.Messages(sessionID); mErr == nil {
		result.Messages = msgs
	}

	// Generate a session name once, after the first user input (proposal §9.2).
	if s.sessions.NeedsAutoName(sessionID) {
		if name := generateSessionName(ctx, provider, model, sessionNameSeed(input.Content, result.Messages)); name != "" {
			if sess, nErr := s.sessions.SetGeneratedName(sessionID, name); nErr == nil {
				result.SessionName = sess.Name
			}
		}
	}

	if err := emitRunStreamEvent(ctx, emit, RunStreamEvent{
		Type:             RunStreamEventFinal,
		Result:           result,
		PendingApprovals: result.PendingApprovals,
	}); err != nil {
		return result, err
	}

	return result, nil
}

func sessionNameSeed(current string, messages []SessionMessage) string {
	if seed := strings.TrimSpace(current); seed != "" {
		return seed
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if seed := strings.TrimSpace(messages[i].Content); seed != "" {
				return seed
			}
		}
	}
	return ""
}

func fallbackRunReply(pending []PendingApproval, traces []RunTrace) string {
	if len(pending) > 0 {
		return "Write approval required before continuing: " + strings.Join(pendingToolNames(pending), ", ")
	}

	for i := len(traces) - 1; i >= 0; i-- {
		trace := traces[i]
		if msg := strings.TrimSpace(trace.Error); msg != "" {
			return "Tool call failed: " + msg
		}

		status, msg := traceResultStatus(trace.Result)
		switch status {
		case "pending_approval":
			if msg == "" {
				msg = trace.Tool
			}
			return "Write approval required before continuing: " + msg
		case "error":
			if msg == "" {
				msg = "tool returned status=error"
			}
			return "Tool call failed: " + msg
		}
	}

	if len(traces) > 0 {
		return "The requested tools ran, but the model did not produce a final reply. Please try again or add more detail."
	}
	return ""
}

func pendingToolNames(pending []PendingApproval) []string {
	names := make([]string, 0, len(pending))
	seen := map[string]struct{}{}
	for _, approval := range pending {
		name := strings.TrimSpace(approval.Tool)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return []string{"write tool"}
	}
	return names
}

func traceResultStatus(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if raw == "error" || raw == "pending_approval" {
		return raw, ""
	}

	var payload struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", ""
	}
	return strings.TrimSpace(payload.Status), strings.TrimSpace(payload.Message)
}

// generateSessionName asks the model for a short title summarizing the first
// user turn. It reuses the vibecoding SDK with no tools and never fails the
// run: on any error it returns an empty string and the caller keeps the
// placeholder name.
func generateSessionName(ctx context.Context, provider settings.AgentProviderConfig, model, firstUserContent string) string {
	prompt := strings.TrimSpace(firstUserContent)
	if prompt == "" {
		return ""
	}

	const namingPrompt = "Generate a short session name for the user's request. " +
		"Reply with only the name, 2-5 words or at most 12 Chinese characters. " +
		"No quotes, no punctuation, no prefixes, no markdown."

	agent, err := agentsdk.NewBuilder().
		WithProviderByName(provider.Vendor, provider.BaseUrl, apiStyle(provider), resolveApiKey(provider.ApiKey)).
		WithModel(model).
		WithMode("agent").
		WithoutBuiltinTools().
		WithSystemPromptExtra(namingPrompt).
		WithMaxIterations(1).
		Build()
	if err != nil {
		return ""
	}

	var title strings.Builder
	for ev := range agent.Run(ctx, prompt) {
		if ev.Type == agentsdk.EventTextDelta {
			title.WriteString(ev.TextDelta)
		}
	}

	return sanitizeTitle(title.String())
}

// sanitizeTitle normalizes an LLM-produced title into a short, clean string.
func sanitizeTitle(raw string) string {
	title := strings.TrimSpace(raw)
	if idx := strings.IndexAny(title, "\r\n"); idx >= 0 {
		title = title[:idx]
	}
	title = strings.Trim(title, " \t\"'`.。")
	if title == "" {
		return ""
	}
	const maxLen = 60
	if len([]rune(title)) > maxLen {
		title = string([]rune(title)[:maxLen])
	}
	return title
}

// fromToolName reverses toolName.
func fromToolName(name string) string {
	return strings.ReplaceAll(name, "__", ".")
}

// persistAudit writes the run's audit entries to the database for replay
// (proposal §8.2 / §8.4). Failures are logged but never fail the run.
func (s *Service) persistAudit(sessionID, project string, entries []AgentAuditEntry) {
	if s == nil || s.app == nil || len(entries) == 0 {
		return
	}
	for _, e := range entries {
		record := &models.AgentAuditRecord{
			SessionID:     sessionID,
			ProjectID:     project,
			Actor:         e.Actor,
			Tool:          e.Tool,
			Category:      e.Category,
			Risk:          e.Risk,
			AuditCategory: e.Audit,
			Decision:      e.Decision,
			Reason:        e.Reason,
			Status:        e.Status,
			ErrorMsg:      e.Error,
		}
		if err := s.app.Dao().SaveAgentAudit(record); err != nil {
			log.Printf("agents: failed to persist audit record: %v", err)
		}
	}
}
