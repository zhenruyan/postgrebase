package agents

import (
	"testing"

	"github.com/zhenruyan/postgrebase/models/settings"
)

func TestProjectConfigPersistenceAndPolicy(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	// default (no row) => inherit
	def := svc.GetProjectConfig("proj-1")
	if def.AllowSchemaChange != "inherit" || def.ApprovalPolicy != "inherit" {
		t.Fatalf("expected inherit defaults, got %+v", def)
	}

	// save overrides
	saved, err := svc.SaveProjectConfig(ProjectConfig{
		Project:           "proj-1",
		DefaultProvider:   "deepseek-main",
		DefaultModel:      "deepseek-chat",
		AllowedTools:      []string{"data.query", "data.insert"},
		AllowSchemaChange: "deny",
		ApprovalPolicy:    "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.DefaultProvider != "deepseek-main" || len(saved.AllowedTools) != 2 {
		t.Fatalf("save did not round-trip: %+v", saved)
	}

	// reload from a fresh service => persisted
	got := NewService(app).GetProjectConfig("proj-1")
	if got.AllowSchemaChange != "deny" || got.ApprovalPolicy != "auto" {
		t.Fatalf("project config not persisted: %+v", got)
	}

	// invalid tri-state is normalized to inherit
	norm, err := svc.SaveProjectConfig(ProjectConfig{Project: "proj-2", AllowSchemaChange: "bogus", ApprovalPolicy: ""})
	if err != nil {
		t.Fatal(err)
	}
	if norm.AllowSchemaChange != "inherit" || norm.ApprovalPolicy != "inherit" {
		t.Fatalf("expected normalized inherit, got %+v", norm)
	}
}

func TestResolvePolicyOverlay(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	// with no project config, policy inherits global (defaults are empty here)
	base := svc.resolvePolicy("proj-x")
	if base.autoApprove {
		t.Fatal("autoApprove should be false by default")
	}

	_, err := svc.SaveProjectConfig(ProjectConfig{
		Project:           "proj-x",
		DefaultProvider:   "p1",
		DefaultModel:      "m1",
		AllowedTools:      []string{"data.query"},
		AllowSchemaChange: "allow",
		ApprovalPolicy:    "auto",
	})
	if err != nil {
		t.Fatal(err)
	}

	p := svc.resolvePolicy("proj-x")
	if p.defaultProvider != "p1" || p.defaultModel != "m1" {
		t.Fatalf("provider/model not applied: %+v", p)
	}
	if !p.allowSchemaChange {
		t.Fatal("allowSchemaChange override (allow) not applied")
	}
	if !p.autoApprove {
		t.Fatal("auto approval policy not applied")
	}
	if len(p.allowedTools) != 1 || p.allowedTools[0] != "data.query" {
		t.Fatalf("allowedTools override not applied: %+v", p.allowedTools)
	}
}

func TestResolveProviderDoesNotFallbackToOtherProvider(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	app.Settings().Agents.Enabled = true
	app.Settings().Agents.DefaultProvider = "missing"
	app.Settings().Agents.DefaultModel = "gpt-4o"
	app.Settings().Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:      "openai-main",
			Vendor:  "openai",
			Api:     "openai-chat",
			Enabled: true,
			Models: []settings.AgentProviderModel{
				{Name: "kimi2.5", ProviderModelId: "kimi2.5", Enabled: true},
			},
		},
	}

	_, _, err := svc.resolveProvider("", "")
	if err == nil {
		t.Fatal("expected error for missing default provider")
	}
}

func TestServiceRefreshReloadsSettingsSnapshot(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	if len(svc.Runtime().Providers) != 0 {
		t.Fatalf("expected empty runtime before config, got %+v", svc.Runtime())
	}

	app.Settings().Agents.Enabled = true
	app.Settings().Agents.DefaultProvider = "gitee-main"
	app.Settings().Agents.Providers = []settings.AgentProviderConfig{
		{Id: "gitee-main", Vendor: "gitee", Api: "openai-chat", Enabled: true},
	}

	svc.Refresh()
	runtime := svc.Runtime()
	if !runtime.Enabled || len(runtime.Providers) != 1 || runtime.Providers[0].Id != "gitee-main" {
		t.Fatalf("runtime snapshot not refreshed: %+v", runtime)
	}
}

func TestResolveProviderPrefersProviderModelsOverGlobalDefault(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	app.Settings().Agents.Enabled = true
	app.Settings().Agents.DefaultModel = "gpt-4o"
	app.Settings().Agents.DefaultProvider = "gitee-main"
	app.Settings().Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:      "gitee-main",
			Vendor:  "gitee",
			Api:     "openai-chat",
			Enabled: true,
			Models: []settings.AgentProviderModel{
				{Name: "kimi2.5", ProviderModelId: "kimi2.5", Enabled: true},
			},
		},
	}
	svc.Refresh()

	provider, model, err := svc.resolveProvider("", "")
	if err != nil {
		t.Fatal(err)
	}
	if provider.Id != "gitee-main" || model != "kimi2.5" {
		t.Fatalf("unexpected resolved provider/model: %+v %s", provider, model)
	}
}

func TestResolveProviderIgnoresStaleUnknownSessionModel(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	app.Settings().Agents.Enabled = true
	app.Settings().Agents.DefaultModel = "gpt-4o"
	app.Settings().Agents.DefaultProvider = "gitee-main"
	app.Settings().Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:      "gitee-main",
			Vendor:  "gitee",
			Api:     "openai-chat",
			Enabled: true,
			Models: []settings.AgentProviderModel{
				{Name: "kimi2.5", ProviderModelId: "kimi2.5", Enabled: true},
			},
		},
	}
	svc.Refresh()

	_, model, err := svc.resolveProvider("gitee-main", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if model != "kimi2.5" {
		t.Fatalf("expected stale model to be ignored, got %q", model)
	}
}

func TestResolveProviderInfersProviderFromModel(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	app.Settings().Agents.Enabled = true
	app.Settings().Agents.DefaultProvider = "openai-main"
	app.Settings().Agents.DefaultModel = "gpt-4o"
	app.Settings().Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:      "gitee-main",
			Vendor:  "gitee",
			Api:     "openai-chat",
			Enabled: true,
			Models: []settings.AgentProviderModel{
				{Name: "kimi2.5", ProviderModelId: "kimi2.5", Enabled: true},
			},
		},
		{
			Id:      "openai-main",
			Vendor:  "openai",
			Api:     "openai-chat",
			Enabled: true,
			Models: []settings.AgentProviderModel{
				{Name: "gpt-4o", ProviderModelId: "gpt-4o", Enabled: true},
			},
		},
	}
	svc.Refresh()

	provider, model, err := svc.resolveProvider("", "kimi2.5")
	if err != nil {
		t.Fatal(err)
	}
	if provider.Id != "gitee-main" || model != "kimi2.5" {
		t.Fatalf("unexpected inferred provider/model: %+v %s", provider, model)
	}
}
