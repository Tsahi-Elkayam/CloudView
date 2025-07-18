package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/providers"
	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
	"github.com/Tsahi-Elkayam/cloudview/test/mocks"
)

// TestInventoryIntegration tests the complete inventory workflow
func TestInventoryIntegration(t *testing.T) {
	// Skip if running in CI without AWS credentials
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration tests")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test with mock provider
	t.Run("mock_provider", func(t *testing.T) {
		testInventoryWithMockProvider(t, ctx, logger)
	})

	// Test with real AWS provider (only if credentials are available)
	if hasAWSCredentials() {
		t.Run("real_aws_provider", func(t *testing.T) {
			testInventoryWithRealAWS(t, ctx, logger)
		})
	} else {
		t.Log("Skipping real AWS test - no credentials available")
	}
}

func testInventoryWithMockProvider(t *testing.T, ctx context.Context, logger *logrus.Logger) {
	// Create mock provider
	mockProvider := mocks.NewMockAWSProvider()
	mockProvider.SetAuthenticated(true)

	// Add some mock resources
	ec2Instance := mocks.CreateMockEC2Instance("i-1234567890abcdef0", "test-instance", "us-east-1", "running")
	s3Bucket := mocks.CreateMockS3Bucket("test-bucket", "us-east-1")
	
	mockProvider.AddResource(ec2Instance)
	mockProvider.AddResource(s3Bucket)

	// Test getting all resources
	filters := types.ResourceFilters{}
	resources, err := mockProvider.GetResources(ctx, filters)
	require.NoError(t, err)
	assert.Len(t, resources, 2)

	// Test filtering by type
	filters.ResourceTypes = []string{"ec2"}
	ec2Resources, err := mockProvider.GetResources(ctx, filters)
	require.NoError(t, err)
	assert.Len(t, ec2Resources, 1)
	assert.Equal(t, "ec2", ec2Resources[0].Type)

	// Test filtering by region
	filters = types.ResourceFilters{
		Regions: []string{"us-east-1"},
	}
	regionResources, err := mockProvider.GetResources(ctx, filters)
	require.NoError(t, err)
	assert.Len(t, regionResources, 2)

	// Test filtering by tags
	filters = types.ResourceFilters{
		Tags: map[string]string{
			"Environment": "test",
		},
	}
	tagResources, err := mockProvider.GetResources(ctx, filters)
	require.NoError(t, err)
	assert.Len(t, tagResources, 1) // Only EC2 instance has Environment=test tag

	// Test getting resource status
	status, err := mockProvider.GetResourceStatus(ctx, "i-1234567890abcdef0")
	require.NoError(t, err)
	assert.Equal(t, "running", status.State)
}

func testInventoryWithRealAWS(t *testing.T, ctx context.Context, logger *logrus.Logger) {
	// Load configuration
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"aws": &config.AWSConfig{
				BaseProviderConfig: config.BaseProviderConfig{
					Enabled: true,
					Regions: []string{"us-east-1"},
				},
				Profile: "default",
				Region:  "us-east-1",
			},
		},
	}

	// Validate configuration
	err := cfg.Validate()
	require.NoError(t, err)

	// Create provider factory
	factory := providers.NewProviderFactory(providers.DefaultRegistry, logger)

	// Create AWS provider
	awsConfig := cfg.Providers["aws"]
	provider, err := factory.CreateProvider(ctx, "aws", awsConfig)
	if err != nil {
		t.Skipf("Failed to create AWS provider (likely auth issue): %v", err)
	}

	// Test basic resource retrieval
	filters := types.ResourceFilters{
		Regions: []string{"us-east-1"},
	}

	// Set a timeout for the API call
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resources, err := provider.GetResources(ctxWithTimeout, filters)
	if err != nil {
		t.Logf("Warning: Failed to get real AWS resources: %v", err)
		return
	}

	t.Logf("Successfully retrieved %d resources from AWS", len(resources))

	// Basic validation of returned resources
	for _, resource := range resources {
		assert.NotEmpty(t, resource.ID)
		assert.NotEmpty(t, resource.Type)
		assert.Equal(t, "aws", resource.Provider)
		assert.NotEmpty(t, resource.Region)
	}
}

func hasAWSCredentials() bool {
	// Check for AWS credentials in environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" {
		return true
	}

	// Check for default profile (this would require file system check)
	// For simplicity, we'll just check environment variables
	return false
}

// TestConfigurationLoading tests configuration loading
func TestConfigurationLoading(t *testing.T) {
	// Test default configuration
	defaultCfg := config.DefaultConfig()
	assert.NotNil(t, defaultCfg)
	assert.Contains(t, defaultCfg.Providers, "aws")

	// Test configuration validation
	err := defaultCfg.Validate()
	assert.NoError(t, err)

	// Test AWS configuration
	awsCfg, ok := defaultCfg.Providers["aws"].(*config.AWSConfig)
	require.True(t, ok)
	assert.Equal(t, "aws", awsCfg.GetProvider())
	assert.Equal(t, "us-east-1", awsCfg.Region)
}

// TestProviderRegistry tests the provider registry functionality
func TestProviderRegistry(t *testing.T) {
	logger := logrus.New()
	registry := providers.NewPluginRegistry(logger)

	// Test registering a mock provider
	mockProvider := mocks.NewMockAWSProvider()
	err := registry.Register(mockProvider)
	assert.NoError(t, err)

	// Test getting the provider
	provider, err := registry.Get("aws")
	assert.NoError(t, err)
	assert.Equal(t, "aws", provider.Name())

	// Test listing providers
	providerNames := registry.List()
	assert.Contains(t, providerNames, "aws")

	// Test provider info
	info := registry.GetProviderInfo()
	assert.Len(t, info, 1)
	assert.Equal(t, "aws", info[0].Name)
}