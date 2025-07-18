package providers

import (
	"context"
	"fmt"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/providers/aws"
	"github.com/sirupsen/logrus"
)

// ProviderFactory creates and manages cloud provider instances
type ProviderFactory struct {
	registry *PluginRegistry
	logger   *logrus.Logger
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(registry *PluginRegistry, logger *logrus.Logger) *ProviderFactory {
	return &ProviderFactory{
		registry: registry,
		logger:   logger,
	}
}

// CreateProvider creates a provider instance with the given configuration
func (f *ProviderFactory) CreateProvider(ctx context.Context, name string, cfg config.ProviderConfig) (CloudProvider, error) {
	f.logger.Debugf("Creating provider: %s", name)
	
	switch name {
	case "aws":
		return f.createAWSProvider(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}
}

// CreateProviders creates multiple provider instances
func (f *ProviderFactory) CreateProviders(ctx context.Context, configs map[string]config.ProviderConfig) (map[string]CloudProvider, error) {
	providers := make(map[string]CloudProvider)
	
	for name, cfg := range configs {
		if !cfg.IsEnabled() {
			f.logger.Debugf("Skipping disabled provider: %s", name)
			continue
		}
		
		provider, err := f.CreateProvider(ctx, name, cfg)
		if err != nil {
			f.logger.Errorf("Failed to create provider %s: %v", name, err)
			continue
		}
		
		providers[name] = provider
	}
	
	return providers, nil
}

// CreateEnabledProviders creates all enabled providers from configuration
func (f *ProviderFactory) CreateEnabledProviders(ctx context.Context, cfg *config.Config) (map[string]CloudProvider, error) {
	return f.CreateProviders(ctx, cfg.Providers)
}

// createAWSProvider creates an AWS provider instance
func (f *ProviderFactory) createAWSProvider(ctx context.Context, cfg config.ProviderConfig) (CloudProvider, error) {
	awsConfig, ok := cfg.(*config.AWSConfig)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type for AWS provider")
	}
	
	provider, err := aws.NewAWSProvider(awsConfig, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS provider: %w", err)
	}
	
	// Authenticate the provider
	if err := provider.Authenticate(ctx, awsConfig); err != nil {
		return nil, fmt.Errorf("failed to authenticate AWS provider: %w", err)
	}
	
	f.logger.Debugf("Successfully created and authenticated AWS provider")
	return provider, nil
}

// ValidateProviderConfig validates a provider configuration
func (f *ProviderFactory) ValidateProviderConfig(name string, cfg config.ProviderConfig) error {
	switch name {
	case "aws":
		awsConfig, ok := cfg.(*config.AWSConfig)
		if !ok {
			return fmt.Errorf("invalid configuration type for AWS provider")
		}
		return awsConfig.Validate()
	default:
		return fmt.Errorf("unsupported provider: %s", name)
	}
}

// GetSupportedProviders returns a list of supported provider names
func (f *ProviderFactory) GetSupportedProviders() []string {
	return []string{"aws"}
}

// DefaultFactory is the global factory instance
var DefaultFactory *ProviderFactory

// init initializes the default factory
func init() {
	DefaultFactory = NewProviderFactory(DefaultRegistry, logrus.New())
}