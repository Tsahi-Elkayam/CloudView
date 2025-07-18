package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	shared "github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// EC2Service handles EC2-related operations
type EC2Service struct {
	client *ec2.Client
	config *config.AWSConfig
	logger *logrus.Logger
}

// NewEC2Service creates a new EC2 service
func NewEC2Service(client *ec2.Client, cfg *config.AWSConfig, logger *logrus.Logger) *EC2Service {
	return &EC2Service{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetInstances retrieves all EC2 instances
func (s *EC2Service) GetInstances(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	var allInstances []models.Resource
	
	// Get regions to query
	regions := s.getRegionsToQuery(filters.Regions)
	
	for _, region := range regions {
		instances, err := s.getInstancesInRegion(ctx, region, filters)
		if err != nil {
			s.logger.Errorf("Failed to get instances in region %s: %v", region, err)
			continue
		}
		allInstances = append(allInstances, instances...)
	}
	
	s.logger.Debugf("Retrieved %d EC2 instances", len(allInstances))
	return allInstances, nil
}

// GetInstanceStatus retrieves the status of a specific EC2 instance
func (s *EC2Service) GetInstanceStatus(ctx context.Context, instanceID string) (*models.ResourceStatus, error) {
	// Try to find the instance in all configured regions
	regions := s.config.GetRegions()
	
	for _, region := range regions {
		// Create a client for this region
		regionClient := s.createRegionClient(region)
		
		input := &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}
		
		result, err := regionClient.DescribeInstances(ctx, input)
		if err != nil {
			continue // Try next region
		}
		
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				if aws.ToString(instance.InstanceId) == instanceID {
					return &models.ResourceStatus{
						State:       string(instance.State.Name),
						Health:      s.mapInstanceHealthToHealth(instance.State.Name),
						LastChecked: time.Now(),
					}, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("EC2 instance %s not found", instanceID)
}

// getInstancesInRegion retrieves instances from a specific region
func (s *EC2Service) getInstancesInRegion(ctx context.Context, region string, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debugf("Getting EC2 instances in region: %s", region)
	
	// Create a client for this region
	regionClient := s.createRegionClient(region)
	
	// Build EC2 filters
	ec2Filters := s.buildEC2Filters(filters)
	
	input := &ec2.DescribeInstancesInput{
		Filters: ec2Filters,
	}
	
	var instances []models.Resource
	
	// Use paginator to handle large result sets
	paginator := ec2.NewDescribeInstancesPaginator(regionClient, input)
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances in region %s: %w", region, err)
		}
		
		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				resource := s.convertInstanceToResource(instance, region)
				
				// Apply additional filters
				if s.matchesFilters(resource, filters) {
					instances = append(instances, *resource)
				}
			}
		}
	}
	
	s.logger.Debugf("Found %d EC2 instances in region %s", len(instances), region)
	return instances, nil
}

// convertInstanceToResource converts an EC2 instance to a Resource model
func (s *EC2Service) convertInstanceToResource(instance types.Instance, region string) *models.Resource {
	// Get instance name from tags
	name := aws.ToString(instance.InstanceId)
	for _, tag := range instance.Tags {
		if aws.ToString(tag.Key) == "Name" {
			name = aws.ToString(tag.Value)
			break
		}
	}
	
	// Convert tags
	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	
	// Create resource
	resource := models.NewResource(
		aws.ToString(instance.InstanceId),
		name,
		string(models.ResourceTypeVirtualMachine),
		"aws",
		region,
	)
	
	// Update status
	resource.UpdateStatus(
		string(instance.State.Name),
		s.mapInstanceHealthToHealth(instance.State.Name),
	)
	
	// Set tags
	resource.Tags = tags
	
	// Set creation time
	if instance.LaunchTime != nil {
		resource.CreatedAt = *instance.LaunchTime
	}
	
	// Add metadata
	resource.SetMetadata("instance_type", string(instance.InstanceType))
	if instance.Platform != "" {
		resource.SetMetadata("platform", string(instance.Platform))
	} else {
		resource.SetMetadata("platform", "linux") // Default for non-Windows instances
	}
	resource.SetMetadata("vpc_id", aws.ToString(instance.VpcId))
	resource.SetMetadata("subnet_id", aws.ToString(instance.SubnetId))
	resource.SetMetadata("availability_zone", aws.ToString(instance.Placement.AvailabilityZone))
	resource.SetMetadata("public_ip", aws.ToString(instance.PublicIpAddress))
	resource.SetMetadata("private_ip", aws.ToString(instance.PrivateIpAddress))
	resource.SetMetadata("image_id", aws.ToString(instance.ImageId))
	resource.SetMetadata("key_name", aws.ToString(instance.KeyName))
	
	// Add security groups
	var securityGroups []string
	for _, sg := range instance.SecurityGroups {
		securityGroups = append(securityGroups, aws.ToString(sg.GroupId))
	}
	resource.SetMetadata("security_groups", securityGroups)
	
	return resource
}

// buildEC2Filters builds EC2 API filters from resource filters
func (s *EC2Service) buildEC2Filters(filters shared.ResourceFilters) []types.Filter {
	var ec2Filters []types.Filter
	
	// Filter by instance state
	if len(filters.Status) > 0 {
		ec2Filters = append(ec2Filters, types.Filter{
			Name:   aws.String("instance-state-name"),
			Values: filters.Status,
		})
	}
	
	// Filter by tags
	for key, value := range filters.Tags {
		ec2Filters = append(ec2Filters, types.Filter{
			Name:   aws.String(fmt.Sprintf("tag:%s", key)),
			Values: []string{value},
		})
	}
	
	return ec2Filters
}

// matchesFilters checks if a resource matches the given filters
func (s *EC2Service) matchesFilters(resource *models.Resource, filters shared.ResourceFilters) bool {
	// Check resource type filter
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if strings.EqualFold(rt, "ec2") || strings.EqualFold(rt, "instance") || strings.EqualFold(rt, "virtual_machine") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check creation time filters
	if filters.CreatedAfter != nil && resource.CreatedAt.Before(*filters.CreatedAfter) {
		return false
	}
	
	if filters.CreatedBefore != nil && resource.CreatedAt.After(*filters.CreatedBefore) {
		return false
	}
	
	return true
}

// mapInstanceHealthToHealth maps EC2 instance state to resource health
func (s *EC2Service) mapInstanceHealthToHealth(state types.InstanceStateName) string {
	switch state {
	case types.InstanceStateNameRunning:
		return string(models.HealthHealthy)
	case types.InstanceStateNameStopped, types.InstanceStateNameStopping:
		return string(models.HealthUnhealthy)
	case types.InstanceStateNamePending:
		return string(models.HealthWarning)
	case types.InstanceStateNameTerminated:
		return string(models.HealthUnhealthy)
	case types.InstanceStateNameShuttingDown:
		return string(models.HealthWarning)
	default:
		return string(models.HealthUnknown)
	}
}

// getRegionsToQuery determines which regions to query based on filters and config
func (s *EC2Service) getRegionsToQuery(filterRegions []string) []string {
	// If specific regions are requested via filters, use those
	if len(filterRegions) > 0 {
		return filterRegions
	}
	
	// If regions are configured, use those
	configRegions := s.config.GetRegions()
	if len(configRegions) > 0 {
		return configRegions
	}
	
	// Fallback to primary region if no regions specified
	if s.config.Region != "" {
		return []string{s.config.Region}
	}
	
	// Ultimate fallback to us-east-1
	return []string{"us-east-1"}
}

// createRegionClient creates an EC2 client for a specific region
func (s *EC2Service) createRegionClient(region string) *ec2.Client {
	// Create a new config with the specific region
	cfg := s.client.Options()
	cfg.Region = region
	
	return ec2.New(cfg)
}