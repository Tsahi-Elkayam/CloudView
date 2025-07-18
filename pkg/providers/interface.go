package providers

import (
	"context"

	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// CloudProvider defines the interface that all cloud provider plugins must implement
type CloudProvider interface {
	// Provider metadata
	Name() string
	Description() string
	SupportedRegions() []string
	
	// Authentication
	Authenticate(ctx context.Context, config config.ProviderConfig) error
	IsAuthenticated() bool
	
	// Resource management
	GetResources(ctx context.Context, filters types.ResourceFilters) ([]models.Resource, error)
	GetResourcesByType(ctx context.Context, resourceType string, filters types.ResourceFilters) ([]models.Resource, error)
	GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error)
	
	// Cost management (for future milestones)
	GetCosts(ctx context.Context, period types.CostPeriod) ([]models.Cost, error)
	GetCostsByService(ctx context.Context, period types.CostPeriod) ([]models.ServiceCost, error)
	GetCostForecast(ctx context.Context, days int) ([]models.CostForecast, error)
	
	// Monitoring and alerts (for future milestones)
	GetAlerts(ctx context.Context, filters types.AlertFilters) ([]models.Alert, error)
	GetMetrics(ctx context.Context, resourceID string, metrics []string) ([]models.Metric, error)
	
	// Security (for future milestones)
	GetSecurityFindings(ctx context.Context, filters types.SecurityFilters) ([]models.SecurityFinding, error)
	GetComplianceStatus(ctx context.Context, framework string) ([]models.ComplianceResult, error)
	
	// Recommendations (for future milestones)
	GetRecommendations(ctx context.Context, categories []string) ([]models.Recommendation, error)
	
	// Utility methods
	ValidateConfig(config config.ProviderConfig) error
	GetSupportedResourceTypes() []string
}

// ProviderResult holds the result of a provider operation
type ProviderResult struct {
	Provider  string
	Resources []models.Resource
	Error     error
}

// RegistrationFunc is the function signature for provider registration
type RegistrationFunc func() CloudProvider