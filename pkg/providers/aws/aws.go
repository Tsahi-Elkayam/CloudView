package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/internal/auth"
	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// AWSProvider implements the CloudProvider interface for AWS
type AWSProvider struct {
	config        *config.AWSConfig
	authenticator *auth.AWSAuthenticator
	awsConfig     aws.Config
	logger        *logrus.Logger
	
	// Service clients
	ec2Service *EC2Service
	s3Service  *S3Service
	iamService *IAMService
	rdsService *RDSService
	vpcService *VPCService
	
	// State
	authenticated bool
	mu            sync.RWMutex
}

// NewAWSProvider creates a new AWS provider instance
func NewAWSProvider(cfg *config.AWSConfig, logger *logrus.Logger) (*AWSProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("AWS configuration cannot be nil")
	}
	
	if logger == nil {
		logger = logrus.New()
	}
	
	authenticator := auth.NewAWSAuthenticator(cfg)
	
	return &AWSProvider{
		config:        cfg,
		authenticator: authenticator,
		logger:        logger,
		authenticated: false,
	}, nil
}

// Name returns the provider name
func (p *AWSProvider) Name() string {
	return "aws"
}

// Description returns the provider description
func (p *AWSProvider) Description() string {
	return "Amazon Web Services (AWS) cloud provider"
}

// SupportedRegions returns the list of supported AWS regions
func (p *AWSProvider) SupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ca-central-1", "sa-east-1", "af-south-1", "me-south-1",
	}
}

// Authenticate authenticates with AWS
func (p *AWSProvider) Authenticate(ctx context.Context, cfg config.ProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	awsConfig, ok := cfg.(*config.AWSConfig)
	if !ok {
		return fmt.Errorf("invalid configuration type, expected *config.AWSConfig")
	}
	
	// Update configuration
	p.config = awsConfig
	p.authenticator = auth.NewAWSAuthenticator(awsConfig)
	
	// Authenticate
	awsCfg, err := p.authenticator.Authenticate(ctx)
	if err != nil {
		p.authenticated = false
		return fmt.Errorf("AWS authentication failed: %w", err)
	}
	
	p.awsConfig = awsCfg
	p.authenticated = true
	
	// Initialize services
	if err := p.initializeServices(); err != nil {
		p.authenticated = false
		return fmt.Errorf("failed to initialize AWS services: %w", err)
	}
	
	// Validate credentials
	identity, err := p.authenticator.ValidateCredentials(ctx)
	if err != nil {
		p.authenticated = false
		return fmt.Errorf("AWS credential validation failed: %w", err)
	}
	
	p.logger.Infof("Successfully authenticated with AWS as %s (Account: %s)", 
		aws.ToString(identity.Arn), 
		aws.ToString(identity.Account))
	
	return nil
}

// IsAuthenticated returns whether the provider is authenticated
func (p *AWSProvider) IsAuthenticated() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.authenticated
}

// GetResources retrieves all resources with the given filters
func (p *AWSProvider) GetResources(ctx context.Context, filters types.ResourceFilters) ([]models.Resource, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("AWS provider is not authenticated")
	}
	
	var allResources []models.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex
	resourceChan := make(chan []models.Resource, 25)
	errorChan := make(chan error, 25)
	
	// Get EC2 instances
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.ec2Service.GetInstances(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get EC2 instances: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get S3 buckets
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.s3Service.GetBuckets(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get S3 buckets: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get RDS databases
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.rdsService.GetDatabases(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get RDS databases: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get RDS clusters
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.rdsService.GetClusters(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get RDS clusters: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get IAM users
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.iamService.GetUsers(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get IAM users: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get IAM roles
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.iamService.GetRoles(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get IAM roles: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get IAM policies
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.iamService.GetPolicies(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get IAM policies: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get VPCs
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.vpcService.GetVPCs(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get VPCs: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Get Security Groups
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := p.vpcService.GetSecurityGroups(ctx, filters)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get security groups: %w", err)
			return
		}
		resourceChan <- resources
	}()
	
	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resourceChan)
		close(errorChan)
	}()
	
	// Collect results
	var errors []error
	for {
		select {
		case resources, ok := <-resourceChan:
			if !ok {
				resourceChan = nil
			} else {
				mu.Lock()
				allResources = append(allResources, resources...)
				mu.Unlock()
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				errors = append(errors, err)
			}
		}
		
		if resourceChan == nil && errorChan == nil {
			break
		}
	}
	
	// Log any errors but don't fail completely
	for _, err := range errors {
		p.logger.Warn(err)
	}
	
	p.logger.Debugf("Retrieved %d resources from AWS", len(allResources))
	return allResources, nil
}

// GetResourcesByType retrieves resources of a specific type
func (p *AWSProvider) GetResourcesByType(ctx context.Context, resourceType string, filters types.ResourceFilters) ([]models.Resource, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("AWS provider is not authenticated")
	}
	
	switch resourceType {
	// EC2 resources
	case "ec2", "virtual_machine", "instance":
		return p.ec2Service.GetInstances(ctx, filters)
	
	// S3 resources
	case "s3", "bucket", "object_storage":
		return p.s3Service.GetBuckets(ctx, filters)
	
	// RDS resources
	case "rds", "rds_instance", "database", "postgres", "postgresql", "mysql":
		return p.rdsService.GetDatabases(ctx, filters)
	case "rds_cluster", "aurora", "cluster":
		return p.rdsService.GetClusters(ctx, filters)
	
	// IAM resources
	case "iam", "iam_user", "user":
		return p.iamService.GetUsers(ctx, filters)
	case "iam_role", "role":
		return p.iamService.GetRoles(ctx, filters)
	case "iam_policy", "policy":
		return p.iamService.GetPolicies(ctx, filters)
	
	// VPC resources  
	case "vpc", "network":
		return p.vpcService.GetVPCs(ctx, filters)
	case "security_group", "firewall", "sg":
		return p.vpcService.GetSecurityGroups(ctx, filters)
	
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// GetResourceStatus retrieves the status of a specific resource
func (p *AWSProvider) GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error) {
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("AWS provider is not authenticated")
	}
	
	// Try to find the resource in EC2
	if status, err := p.ec2Service.GetInstanceStatus(ctx, resourceID); err == nil {
		return status, nil
	}
	
	// Try to find the resource in S3
	if status, err := p.s3Service.GetBucketStatus(ctx, resourceID); err == nil {
		return status, nil
	}
	
	// For IAM and RDS, we can assume they're active if they exist
	// (We'd need separate status checking methods for more detailed status)
	
	return nil, fmt.Errorf("resource %s not found", resourceID)
}

// ValidateConfig validates the AWS configuration
func (p *AWSProvider) ValidateConfig(cfg config.ProviderConfig) error {
	awsConfig, ok := cfg.(*config.AWSConfig)
	if !ok {
		return fmt.Errorf("invalid configuration type, expected *config.AWSConfig")
	}
	
	return awsConfig.Validate()
}

// GetSupportedResourceTypes returns the list of supported resource types
func (p *AWSProvider) GetSupportedResourceTypes() []string {
	return []string{
		// EC2 resources
		"ec2", "instance", "virtual_machine",
		
		// S3 resources
		"s3", "bucket", "object_storage",
		
		// RDS resources
		"rds", "rds_instance", "rds_cluster", "database", 
		"postgres", "postgresql", "mysql", "aurora", "cluster",
		
		// IAM resources
		"iam", "iam_user", "iam_role", "iam_policy",
		"user", "role", "policy",
		
		// VPC resources
		"vpc", "network", "security_group", "firewall", "sg",
	}
}

// initializeServices initializes AWS service clients
func (p *AWSProvider) initializeServices() error {
	// Initialize EC2 service
	ec2Client := ec2.NewFromConfig(p.awsConfig)
	p.ec2Service = NewEC2Service(ec2Client, p.config, p.logger)
	
	// Initialize S3 service
	s3Client := s3.NewFromConfig(p.awsConfig)
	p.s3Service = NewS3Service(s3Client, p.config, p.logger)
	
	// Initialize IAM service
	iamClient := iam.NewFromConfig(p.awsConfig)
	p.iamService = NewIAMService(iamClient, p.config, p.logger)
	
	// Initialize RDS service
	rdsClient := rds.NewFromConfig(p.awsConfig)
	p.rdsService = NewRDSService(rdsClient, p.config, p.logger)
	
	// Initialize VPC service (uses EC2 client)
	p.vpcService = NewVPCService(ec2Client, p.config, p.logger)
	
	return nil
}

// Placeholder implementations for future milestones
func (p *AWSProvider) GetCosts(ctx context.Context, period types.CostPeriod) ([]models.Cost, error) {
	return nil, fmt.Errorf("cost management not implemented yet")
}

func (p *AWSProvider) GetCostsByService(ctx context.Context, period types.CostPeriod) ([]models.ServiceCost, error) {
	return nil, fmt.Errorf("cost management not implemented yet")
}

func (p *AWSProvider) GetCostForecast(ctx context.Context, days int) ([]models.CostForecast, error) {
	return nil, fmt.Errorf("cost management not implemented yet")
}

func (p *AWSProvider) GetAlerts(ctx context.Context, filters types.AlertFilters) ([]models.Alert, error) {
	return nil, fmt.Errorf("alert management not implemented yet")
}

func (p *AWSProvider) GetMetrics(ctx context.Context, resourceID string, metrics []string) ([]models.Metric, error) {
	return nil, fmt.Errorf("metrics not implemented yet")
}

func (p *AWSProvider) GetSecurityFindings(ctx context.Context, filters types.SecurityFilters) ([]models.SecurityFinding, error) {
	return nil, fmt.Errorf("security findings not implemented yet")
}

func (p *AWSProvider) GetComplianceStatus(ctx context.Context, framework string) ([]models.ComplianceResult, error) {
	return nil, fmt.Errorf("compliance status not implemented yet")
}

func (p *AWSProvider) GetRecommendations(ctx context.Context, categories []string) ([]models.Recommendation, error) {
	return nil, fmt.Errorf("recommendations not implemented yet")
}