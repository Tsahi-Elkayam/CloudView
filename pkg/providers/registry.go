package providers

import (
	"fmt"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

// PluginRegistry manages all registered cloud provider plugins
type PluginRegistry struct {
	providers map[string]CloudProvider
	mu        sync.RWMutex
	logger    *logrus.Logger
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(logger *logrus.Logger) *PluginRegistry {
	return &PluginRegistry{
		providers: make(map[string]CloudProvider),
		logger:    logger,
	}
}

// Register registers a new cloud provider plugin
func (r *PluginRegistry) Register(provider CloudProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	r.logger.Debugf("Registered provider: %s", name)

	return nil
}

// Unregister removes a provider from the registry
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	delete(r.providers, name)
	r.logger.Debugf("Unregistered provider: %s", name)

	return nil
}

// Get retrieves a provider by name
func (r *PluginRegistry) Get(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// List returns a list of all registered provider names
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetAll returns all registered providers
func (r *PluginRegistry) GetAll() map[string]CloudProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]CloudProvider)
	for name, provider := range r.providers {
		result[name] = provider
	}

	return result
}

// Exists checks if a provider is registered
func (r *PluginRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[name]
	return exists
}

// Count returns the number of registered providers
func (r *PluginRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers)
}

// GetProviderInfo returns detailed information about all providers
func (r *PluginRegistry) GetProviderInfo() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]ProviderInfo, 0, len(r.providers))
	for name, provider := range r.providers {
		info = append(info, ProviderInfo{
			Name:             name,
			Description:      provider.Description(),
			SupportedRegions: provider.SupportedRegions(),
			ResourceTypes:    provider.GetSupportedResourceTypes(),
			IsAuthenticated:  provider.IsAuthenticated(),
		})
	}

	return info
}

// ProviderInfo holds information about a registered provider
type ProviderInfo struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	SupportedRegions []string `json:"supported_regions"`
	ResourceTypes    []string `json:"resource_types"`
	IsAuthenticated  bool     `json:"is_authenticated"`
}

// DefaultRegistry is the global registry instance
var DefaultRegistry *PluginRegistry

// init initializes the default registry
func init() {
	DefaultRegistry = NewPluginRegistry(logrus.New())
}
