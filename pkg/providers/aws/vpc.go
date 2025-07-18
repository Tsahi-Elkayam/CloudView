package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	shared "github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// VPCService handles VPC and networking-related operations
type VPCService struct {
	client *ec2.Client
	config *config.AWSConfig
	logger *logrus.Logger
}

// NewVPCService creates a new VPC service
func NewVPCService(client *ec2.Client, cfg *config.AWSConfig, logger *logrus.Logger) *VPCService {
	return &VPCService{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetVPCs retrieves all VPCs
func (s *VPCService) GetVPCs(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	var allVPCs []models.Resource
	
	// Get regions to query
	regions := s.getRegionsToQuery(filters.Regions)
	
	for _, region := range regions {
		vpcs, err := s.getVPCsInRegion(ctx, region, filters)
		if err != nil {
			s.logger.Errorf("Failed to get VPCs in region %s: %v", region, err)
			continue
		}
		allVPCs = append(allVPCs, vpcs...)
	}
	
	s.logger.Debugf("Retrieved %d VPCs", len(allVPCs))
	return allVPCs, nil
}

// GetSecurityGroups retrieves all security groups
func (s *VPCService) GetSecurityGroups(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	var allSecurityGroups []models.Resource
	
	// Get regions to query
	regions := s.getRegionsToQuery(filters.Regions)
	
	for _, region := range regions {
		securityGroups, err := s.getSecurityGroupsInRegion(ctx, region, filters)
		if err != nil {
			s.logger.Errorf("Failed to get security groups in region %s: %v", region, err)
			continue
		}
		allSecurityGroups = append(allSecurityGroups, securityGroups...)
	}
	
	s.logger.Debugf("Retrieved %d security groups", len(allSecurityGroups))
	return allSecurityGroups, nil
}

// getVPCsInRegion retrieves VPCs from a specific region
func (s *VPCService) getVPCsInRegion(ctx context.Context, region string, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debugf("Getting VPCs in region: %s", region)
	
	// Create a client for this region
	regionClient := s.createRegionClient(region)
	
	var vpcs []models.Resource
	
	// Use paginator to handle large result sets
	paginator := ec2.NewDescribeVpcsPaginator(regionClient, &ec2.DescribeVpcsInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe VPCs in region %s: %w", region, err)
		}
		
		for _, vpc := range page.Vpcs {
			resource := s.convertVPCToResource(vpc, region)
			
			// Apply additional filters
			if s.matchesFilters(resource, filters) {
				vpcs = append(vpcs, *resource)
			}
		}
	}
	
	s.logger.Debugf("Found %d VPCs in region %s", len(vpcs), region)
	return vpcs, nil
}

// getSecurityGroupsInRegion retrieves security groups from a specific region
func (s *VPCService) getSecurityGroupsInRegion(ctx context.Context, region string, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debugf("Getting security groups in region: %s", region)
	
	// Create a client for this region
	regionClient := s.createRegionClient(region)
	
	var securityGroups []models.Resource
	
	// Use paginator to handle large result sets
	paginator := ec2.NewDescribeSecurityGroupsPaginator(regionClient, &ec2.DescribeSecurityGroupsInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe security groups in region %s: %w", region, err)
		}
		
		for _, sg := range page.SecurityGroups {
			resource := s.convertSecurityGroupToResource(sg, region)
			
			// Apply additional filters
			if s.matchesFilters(resource, filters) {
				securityGroups = append(securityGroups, *resource)
			}
		}
	}
	
	s.logger.Debugf("Found %d security groups in region %s", len(securityGroups), region)
	return securityGroups, nil
}

// convertVPCToResource converts a VPC to a Resource model
func (s *VPCService) convertVPCToResource(vpc types.Vpc, region string) *models.Resource {
	// Get VPC name from tags
	name := aws.ToString(vpc.VpcId)
	for _, tag := range vpc.Tags {
		if aws.ToString(tag.Key) == "Name" {
			name = aws.ToString(tag.Value)
			break
		}
	}
	
	resource := models.NewResource(
		aws.ToString(vpc.VpcId),
		name,
		"vpc",
		"aws",
		region,
	)
	
	// Update status
	resource.UpdateStatus(
		string(vpc.State),
		s.mapVPCStateToHealth(vpc.State),
	)
	
	// Convert tags
	tags := make(map[string]string)
	for _, tag := range vpc.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	resource.Tags = tags
	
	// Add metadata
	resource.SetMetadata("cidr_block", aws.ToString(vpc.CidrBlock))
	resource.SetMetadata("dhcp_options_id", aws.ToString(vpc.DhcpOptionsId))
	resource.SetMetadata("instance_tenancy", string(vpc.InstanceTenancy))
	resource.SetMetadata("is_default", vpc.IsDefault)
	resource.SetMetadata("owner_id", aws.ToString(vpc.OwnerId))
	
	// Add IPv6 CIDR blocks if present
	if len(vpc.Ipv6CidrBlockAssociationSet) > 0 {
		var ipv6Blocks []string
		for _, block := range vpc.Ipv6CidrBlockAssociationSet {
			ipv6Blocks = append(ipv6Blocks, aws.ToString(block.Ipv6CidrBlock))
		}
		resource.SetMetadata("ipv6_cidr_blocks", ipv6Blocks)
	}
	
	return resource
}

// convertSecurityGroupToResource converts a security group to a Resource model
func (s *VPCService) convertSecurityGroupToResource(sg types.SecurityGroup, region string) *models.Resource {
	resource := models.NewResource(
		aws.ToString(sg.GroupId),
		aws.ToString(sg.GroupName),
		"security_group",
		"aws",
		region,
	)
	
	resource.UpdateStatus("available", string(models.HealthHealthy))
	
	// Convert tags
	tags := make(map[string]string)
	for _, tag := range sg.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	resource.Tags = tags
	
	// Add metadata
	resource.SetMetadata("description", aws.ToString(sg.Description))
	resource.SetMetadata("vpc_id", aws.ToString(sg.VpcId))
	resource.SetMetadata("owner_id", aws.ToString(sg.OwnerId))
	
	// Add ingress rules
	var ingressRules []map[string]interface{}
	for _, rule := range sg.IpPermissions {
		ruleMap := map[string]interface{}{
			"protocol":    aws.ToString(rule.IpProtocol),
			"from_port":   rule.FromPort,
			"to_port":     rule.ToPort,
		}
		
		// Add IP ranges
		var ipRanges []string
		for _, ipRange := range rule.IpRanges {
			ipRanges = append(ipRanges, aws.ToString(ipRange.CidrIp))
		}
		ruleMap["ip_ranges"] = ipRanges
		
		// Add security group references
		var sgReferences []string
		for _, sgRef := range rule.UserIdGroupPairs {
			sgReferences = append(sgReferences, aws.ToString(sgRef.GroupId))
		}
		ruleMap["security_groups"] = sgReferences
		
		ingressRules = append(ingressRules, ruleMap)
	}
	resource.SetMetadata("ingress_rules", ingressRules)
	
	// Add egress rules
	var egressRules []map[string]interface{}
	for _, rule := range sg.IpPermissionsEgress {
		ruleMap := map[string]interface{}{
			"protocol":    aws.ToString(rule.IpProtocol),
			"from_port":   rule.FromPort,
			"to_port":     rule.ToPort,
		}
		
		// Add IP ranges
		var ipRanges []string
		for _, ipRange := range rule.IpRanges {
			ipRanges = append(ipRanges, aws.ToString(ipRange.CidrIp))
		}
		ruleMap["ip_ranges"] = ipRanges
		
		egressRules = append(egressRules, ruleMap)
	}
	resource.SetMetadata("egress_rules", egressRules)
	
	return resource
}

// mapVPCStateToHealth maps VPC state to resource health
func (s *VPCService) mapVPCStateToHealth(state types.VpcState) string {
	switch state {
	case types.VpcStateAvailable:
		return string(models.HealthHealthy)
	case types.VpcStatePending:
		return string(models.HealthWarning)
	default:
		return string(models.HealthUnknown)
	}
}

// matchesFilters checks if a resource matches the given filters
func (s *VPCService) matchesFilters(resource *models.Resource, filters shared.ResourceFilters) bool {
	// Check resource type filter
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if strings.EqualFold(rt, "vpc") || 
			   strings.EqualFold(rt, "network") ||
			   strings.EqualFold(rt, "security_group") ||
			   strings.EqualFold(rt, "firewall") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check region filter
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
	
	// Check tag filters
	for key, value := range filters.Tags {
		if resourceValue, exists := resource.GetTag(key); !exists || resourceValue != value {
			return false
		}
	}
	
	return true
}

// getRegionsToQuery determines which regions to query based on filters and config
func (s *VPCService) getRegionsToQuery(filterRegions []string) []string {
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
func (s *VPCService) createRegionClient(region string) *ec2.Client {
	// Create a new config with the specific region
	cfg := s.client.Options()
	cfg.Region = region
	
	return ec2.New(cfg)
}