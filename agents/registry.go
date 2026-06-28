package agents

import (
	"errors"
	"sort"
	"strings"

	"github.com/zhenruyan/postgrebase/models/settings"
)

// Runtime describes the embedded agent surface exposed by PostgreBase.
type Runtime struct {
	Enabled           bool       `json:"enabled"`
	DefaultProvider   string     `json:"defaultProvider"`
	DefaultModel      string     `json:"defaultModel"`
	AllowSchemaChange bool       `json:"allowSchemaChange"`
	AllowedTools      []string   `json:"allowedTools"`
	Providers         []Provider `json:"providers"`
}

// Provider is a provider-level view derived from stored settings.
type Provider struct {
	Id           string  `json:"id"`
	Vendor       string  `json:"vendor"`
	BaseUrl      string  `json:"baseUrl"`
	Enabled      bool    `json:"enabled"`
	DefaultModel string  `json:"defaultModel"`
	Models       []Model `json:"models"`
}

// Model is a model-level view derived from stored settings.
type Model struct {
	Name             string `json:"name"`
	ProviderModelId  string `json:"providerModelId"`
	SupportsVision   bool   `json:"supportsVision"`
	SupportsToolUse  bool   `json:"supportsToolUse"`
	SupportsDocument bool   `json:"supportsDocument"`
	Enabled          bool   `json:"enabled"`
}

// Registry exposes a static snapshot of agent runtime configuration.
type Registry struct {
	runtime Runtime
}

// NewRegistry builds a registry from app settings.
func NewRegistry(cfg settings.AgentConfig) *Registry {
	reg := &Registry{
		runtime: Runtime{
			Enabled:           cfg.Enabled,
			DefaultProvider:   cfg.DefaultProvider,
			DefaultModel:      cfg.DefaultModel,
			AllowSchemaChange: cfg.AllowSchemaChange,
			AllowedTools:      append([]string(nil), cfg.AllowedTools...),
		},
	}

	reg.runtime.Providers = make([]Provider, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		models := make([]Model, 0, len(p.Models))
		for _, m := range p.Models {
			models = append(models, Model{
				Name:             m.Name,
				ProviderModelId:  m.ProviderModelId,
				SupportsVision:   m.SupportsVision,
				SupportsToolUse:  m.SupportsToolUse,
				SupportsDocument: m.SupportsDocument,
				Enabled:          m.Enabled,
			})
		}

		reg.runtime.Providers = append(reg.runtime.Providers, Provider{
			Id:           p.Id,
			Vendor:       p.Vendor,
			BaseUrl:      p.BaseUrl,
			Enabled:      p.Enabled,
			DefaultModel: p.DefaultModel,
			Models:       models,
		})
	}

	sort.SliceStable(reg.runtime.Providers, func(i, j int) bool {
		return strings.Compare(reg.runtime.Providers[i].Id, reg.runtime.Providers[j].Id) < 0
	})

	return reg
}

// Snapshot returns the registry runtime snapshot.
func (r *Registry) Snapshot() Runtime {
	if r == nil {
		return Runtime{}
	}

	snap := r.runtime
	snap.AllowedTools = append([]string(nil), snap.AllowedTools...)
	snap.Providers = append([]Provider(nil), snap.Providers...)
	return snap
}

// Provider returns a configured provider by id.
func (r *Registry) Provider(id string) (Provider, error) {
	for _, provider := range r.runtime.Providers {
		if provider.Id == id {
			return provider, nil
		}
	}
	return Provider{}, errors.New("agent provider not found")
}
