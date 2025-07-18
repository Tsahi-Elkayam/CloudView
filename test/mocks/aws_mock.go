package mocks

import (
	"context"
	"fmt"
	"time"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// ErrResourceNotFound is returned when a resource is not found
var ErrResourceNotFound = fmt.Errorf("resource not found")

// MockAWSProvider implements the CloudProvider interface for testing
type MockAWSProvider struct {
	authenticated bool
	resources     []models.Resource
	errors        map[string]error
}

// NewMockAWSProvider creates a new mock AWS provider
func NewMockAWSProvider() *MockAWSProvider {
	return &MockAWSProvider{
		authenticated: false,
		resources:     []models.Resource{},
		errors:        make(map[string]error),
	}
}

// SetAuthenticated sets the authentication status
func (m *MockAWSProvider) SetAuthenticated(authenticated bool) {
	m.authenticated = authenticated
}

// AddResource adds a resource to the mock provider
func (m *MockAWSProvider) AddResource(resource models.Resource) {
	m.resources = append(m.resources, resource)
}

// SetError sets an error for a specific method
func (m *MockAWSProvider) SetError(method string, err error) {
	m.errors[method] = err
}

// CloudProvider interface implementation

func (m *MockAWSProvider) Name() string {
	return "aws"
}

func (m *MockAWSProvider) Description() string {
	return "Mock AWS provider for testing"
}

func (m *MockAWSProvider) SupportedRegions() []string {
	return []string{"us-east-1", "us-west-2", "eu-west-1"}
}

func (m *MockAWSProvider) Authenticate(ctx context.Context, config config.ProviderConfig) error {
	if err, exists := m.errors["Authenticate"]; exists {
		return err
	}
	m.authenticated = true
	return nil
}

func (m *MockAWSProvider) IsAuthenticated() bool {
	return m.authenticated
}

func (m *MockAWSProvider) GetResources(ctx context.Context, filters types.ResourceFilters) ([]models.Resource, error) {
	if err, exists := m.errors["GetResources"]; exists {
		return nil, err
	}
	
	// Apply basic filtering
	var filteredResources []models.Resource
	for _, resource := range m.resources {
		if m.matchesFilters(resource, filters) {
			filteredResources = append(filteredResources, resource)
		}
	}
	
	return filteredResources, nil
}

func (m *MockAWSProvider) GetResourcesByType(ctx context.Context, resourceType string, filters types.ResourceFilters) ([]models.Resource, error) {
	if err, exists := m.errors["GetResourcesByType"]; exists {
		return nil, err
	}
	
	var filteredResources []models.Resource
	for _, resource := range m.resources {
		if resource.Type == resourceType && m.matchesFilters(resource, filters) {
			filteredResources = append(filteredResources, resource)
		}
	}
	
	return filteredResources, nil
}

func (m *MockAWSProvider) GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error) {
	if err, exists := m.errors["GetResourceStatus"]; exists {
		return nil, err
	}
	
	for _, resource := range m.resources {
		if resource.ID == resourceID {
			return &resource.Status, nil
		}
	}
	
	return nil, ErrResourceNotFound
}

func (m *MockAWSProvider) ValidateConfig(config config.ProviderConfig) error {
	if err, exists := m.errors["ValidateConfig"]; exists {
		return err
	}
	return nil
}

func (m *MockAWSProvider) GetSupportedResourceTypes() []string {
	return []string{"ec2", "s3", "lambda"}
}

// matchesFilters checks if a resource matches the given filters
func (m *MockAWSProvider) matchesFilters(resource models.Resource, filters types.ResourceFilters) bool {
	// Check regions
	if len(filters.Regions) > 0 {
		found := false
		for _, region := range filters.Regions {
			if resource.Region == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check resource types
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if resource.Type == rt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check tags
	for key, value := range filters.Tags {
		if resourceValue, exists := resource.Tags[key]; !exists || resourceValue != value {
			return false
		}
	}
	
	// Check status
	if len(filters.Status) > 0 {
		found := false
		for _, status := range filters.Status {
			if resource.Status.State == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check creation time
	if filters.CreatedAfter != nil && resource.CreatedAt.Before(*filters.CreatedAfter) {
		return false
	}
	
	if filters.CreatedBefore != nil && resource.CreatedAt.After(*filters.CreatedBefore) {
		return false
	}
	
	return true
}

// Placeholder implementations for future features

func (m *MockAWSProvider) GetCosts(ctx context.Context, period types.CostPeriod) ([]models.Cost, error) {
	if err, exists := m.errors["GetCosts"]; exists {
		return nil, err
	}
	return []models.Cost{}, nil
}

func (m *MockAWSProvider) GetCostsByService(ctx context.Context, period types.CostPeriod) ([]models.ServiceCost, error) {
	if err, exists := m.errors["GetCostsByService"]; exists {
		return nil, err
	}
	return []models.ServiceCost{}, nil
}

func (m *MockAWSProvider) GetCostForecast(ctx context.Context, days int) ([]models.CostForecast, error) {
	if err, exists := m.errors["GetCostForecast"]; exists {
		return nil, err
	}
	return []models.CostForecast{}, nil
}

func (m *MockAWSProvider) GetAlerts(ctx context.Context, filters types.AlertFilters) ([]models.Alert, error) {
	if err, exists := m.errors["GetAlerts"]; exists {
		return nil, err
	}
	return []models.Alert{}, nil
}

func (m *MockAWSProvider) GetMetrics(ctx context.Context, resourceID string, metrics []string) ([]models.Metric, error) {
	if err, exists := m.errors["GetMetrics"]; exists {
		return nil, err
	}
	return []models.Metric{}, nil
}

func (m *MockAWSProvider) GetSecurityFindings(ctx context.Context, filters types.SecurityFilters) ([]models.SecurityFinding, error) {
	if err, exists := m.errors["GetSecurityFindings"]; exists {
		return nil, err
	}
	return []models.SecurityFinding{}, nil
}

func (m *MockAWSProvider) GetComplianceStatus(ctx context.Context, framework string) ([]models.ComplianceResult, error) {
	if err, exists := m.errors["GetComplianceStatus"]; exists {
		return nil, err
	}
	return []models.ComplianceResult{}, nil
}

func (m *MockAWSProvider) GetRecommendations(ctx context.Context, categories []string) ([]models.Recommendation, error) {
	if err, exists := m.errors["GetRecommendations"]; exists {
		return nil, err
	}
	return []models.Recommendation{}, nil
}

// Helper functions for testing

// CreateMockEC2Instance creates a mock EC2 instance
func CreateMockEC2Instance(id, name, region, state string) models.Resource {
	resource := models.NewResource(id, name, "ec2", "aws", region)
	resource.UpdateStatus(state, "healthy")
	resource.SetTag("Environment", "test")
	resource.SetMetadata("instance_type", "t3.micro")
	resource.CreatedAt = time.Now().Add(-24 * time.Hour)
	return *resource
}

// CreateMockS3Bucket creates a mock S3 bucket
func CreateMockS3Bucket(name, region string) models.Resource {
	resource := models.NewResource(name, name, "s3", "aws", region)
	resource.UpdateStatus("available", "healthy")
	resource.SetTag("Purpose", "testing")
	resource.SetMetadata("versioning", "enabled")
	resource.CreatedAt = time.Now().Add(-48 * time.Hour)
	return *resource
}