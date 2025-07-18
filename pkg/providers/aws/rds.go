package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	shared "github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// RDSService handles RDS-related operations
type RDSService struct {
	client *rds.Client
	config *config.AWSConfig
	logger *logrus.Logger
}

// NewRDSService creates a new RDS service
func NewRDSService(client *rds.Client, cfg *config.AWSConfig, logger *logrus.Logger) *RDSService {
	return &RDSService{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetDatabases retrieves all RDS database instances
func (s *RDSService) GetDatabases(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	var allDatabases []models.Resource
	
	// Get regions to query
	regions := s.getRegionsToQuery(filters.Regions)
	
	for _, region := range regions {
		databases, err := s.getDatabasesInRegion(ctx, region, filters)
		if err != nil {
			s.logger.Errorf("Failed to get databases in region %s: %v", region, err)
			continue
		}
		allDatabases = append(allDatabases, databases...)
	}
	
	s.logger.Debugf("Retrieved %d RDS databases", len(allDatabases))
	return allDatabases, nil
}

// GetClusters retrieves all RDS clusters (Aurora)
func (s *RDSService) GetClusters(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	var allClusters []models.Resource
	
	// Get regions to query
	regions := s.getRegionsToQuery(filters.Regions)
	
	for _, region := range regions {
		clusters, err := s.getClustersInRegion(ctx, region, filters)
		if err != nil {
			s.logger.Errorf("Failed to get clusters in region %s: %v", region, err)
			continue
		}
		allClusters = append(allClusters, clusters...)
	}
	
	s.logger.Debugf("Retrieved %d RDS clusters", len(allClusters))
	return allClusters, nil
}

// getDatabasesInRegion retrieves databases from a specific region
func (s *RDSService) getDatabasesInRegion(ctx context.Context, region string, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debugf("Getting RDS databases in region: %s", region)
	
	// Create a client for this region
	regionClient := s.createRegionClient(region)
	
	var databases []models.Resource
	
	// Use paginator to handle large result sets
	paginator := rds.NewDescribeDBInstancesPaginator(regionClient, &rds.DescribeDBInstancesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances in region %s: %w", region, err)
		}
		
		for _, instance := range page.DBInstances {
			resource := s.convertDBInstanceToResource(instance, region)
			
			// Apply additional filters
			if s.matchesFilters(resource, filters) {
				databases = append(databases, *resource)
			}
		}
	}
	
	s.logger.Debugf("Found %d RDS databases in region %s", len(databases), region)
	return databases, nil
}

// getClustersInRegion retrieves clusters from a specific region
func (s *RDSService) getClustersInRegion(ctx context.Context, region string, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debugf("Getting RDS clusters in region: %s", region)
	
	// Create a client for this region
	regionClient := s.createRegionClient(region)
	
	var clusters []models.Resource
	
	// Use paginator to handle large result sets
	paginator := rds.NewDescribeDBClustersPaginator(regionClient, &rds.DescribeDBClustersInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB clusters in region %s: %w", region, err)
		}
		
		for _, cluster := range page.DBClusters {
			resource := s.convertDBClusterToResource(cluster, region)
			
			// Apply additional filters
			if s.matchesFilters(resource, filters) {
				clusters = append(clusters, *resource)
			}
		}
	}
	
	s.logger.Debugf("Found %d RDS clusters in region %s", len(clusters), region)
	return clusters, nil
}

// convertDBInstanceToResource converts an RDS instance to a Resource model
func (s *RDSService) convertDBInstanceToResource(instance types.DBInstance, region string) *models.Resource {
	// Get instance name from identifier
	name := aws.ToString(instance.DBInstanceIdentifier)
	
	resource := models.NewResource(
		aws.ToString(instance.DBInstanceIdentifier),
		name,
		"rds_instance",
		"aws",
		region,
	)
	
	// Update status
	status := aws.ToString(instance.DBInstanceStatus)
	health := s.mapDBStatusToHealth(status)
	resource.UpdateStatus(status, health)
	
	// Set creation time
	if instance.InstanceCreateTime != nil {
		resource.CreatedAt = *instance.InstanceCreateTime
	}
	
	// Add metadata
	resource.SetMetadata("engine", aws.ToString(instance.Engine))
	resource.SetMetadata("engine_version", aws.ToString(instance.EngineVersion))
	resource.SetMetadata("db_instance_class", aws.ToString(instance.DBInstanceClass))
	resource.SetMetadata("master_username", aws.ToString(instance.MasterUsername))
	resource.SetMetadata("database_name", aws.ToString(instance.DBName))
	resource.SetMetadata("allocated_storage", instance.AllocatedStorage)
	resource.SetMetadata("storage_type", aws.ToString(instance.StorageType))
	resource.SetMetadata("multi_az", instance.MultiAZ)
	resource.SetMetadata("publicly_accessible", instance.PubliclyAccessible)
	resource.SetMetadata("storage_encrypted", instance.StorageEncrypted)
	resource.SetMetadata("backup_retention_period", instance.BackupRetentionPeriod)
	resource.SetMetadata("preferred_backup_window", aws.ToString(instance.PreferredBackupWindow))
	resource.SetMetadata("preferred_maintenance_window", aws.ToString(instance.PreferredMaintenanceWindow))
	
	// Endpoint information
	if instance.Endpoint != nil {
		resource.SetMetadata("endpoint_address", aws.ToString(instance.Endpoint.Address))
		resource.SetMetadata("endpoint_port", instance.Endpoint.Port)
	}
	
	// VPC information
	if instance.DBSubnetGroup != nil {
		resource.SetMetadata("vpc_id", aws.ToString(instance.DBSubnetGroup.VpcId))
		resource.SetMetadata("subnet_group", aws.ToString(instance.DBSubnetGroup.DBSubnetGroupName))
	}
	
	// Security groups
	var securityGroups []string
	for _, sg := range instance.VpcSecurityGroups {
		securityGroups = append(securityGroups, aws.ToString(sg.VpcSecurityGroupId))
	}
	resource.SetMetadata("security_groups", securityGroups)
	
	// Tags
	if len(instance.TagList) > 0 {
		for _, tag := range instance.TagList {
			resource.SetTag(aws.ToString(tag.Key), aws.ToString(tag.Value))
		}
	}
	
	return resource
}

// convertDBClusterToResource converts an RDS cluster to a Resource model
func (s *RDSService) convertDBClusterToResource(cluster types.DBCluster, region string) *models.Resource {
	// Get cluster name from identifier
	name := aws.ToString(cluster.DBClusterIdentifier)
	
	resource := models.NewResource(
		aws.ToString(cluster.DBClusterIdentifier),
		name,
		"rds_cluster",
		"aws",
		region,
	)
	
	// Update status
	status := aws.ToString(cluster.Status)
	health := s.mapDBStatusToHealth(status)
	resource.UpdateStatus(status, health)
	
	// Set creation time
	if cluster.ClusterCreateTime != nil {
		resource.CreatedAt = *cluster.ClusterCreateTime
	}
	
	// Add metadata
	resource.SetMetadata("engine", aws.ToString(cluster.Engine))
	resource.SetMetadata("engine_version", aws.ToString(cluster.EngineVersion))
	resource.SetMetadata("engine_mode", aws.ToString(cluster.EngineMode))
	resource.SetMetadata("master_username", aws.ToString(cluster.MasterUsername))
	resource.SetMetadata("database_name", aws.ToString(cluster.DatabaseName))
	resource.SetMetadata("storage_encrypted", cluster.StorageEncrypted)
	resource.SetMetadata("backup_retention_period", cluster.BackupRetentionPeriod)
	resource.SetMetadata("preferred_backup_window", aws.ToString(cluster.PreferredBackupWindow))
	resource.SetMetadata("preferred_maintenance_window", aws.ToString(cluster.PreferredMaintenanceWindow))
	
	// Endpoint information
	resource.SetMetadata("endpoint", aws.ToString(cluster.Endpoint))
	resource.SetMetadata("reader_endpoint", aws.ToString(cluster.ReaderEndpoint))
	resource.SetMetadata("port", cluster.Port)
	
	// Cluster members
	var members []string
	for _, member := range cluster.DBClusterMembers {
		members = append(members, aws.ToString(member.DBInstanceIdentifier))
	}
	resource.SetMetadata("cluster_members", members)
	
	// VPC information
	if cluster.DBSubnetGroup != nil && aws.ToString(cluster.DBSubnetGroup) != "" {
		resource.SetMetadata("subnet_group", aws.ToString(cluster.DBSubnetGroup))
	}
	
	// Security groups
	var securityGroups []string
	for _, sg := range cluster.VpcSecurityGroups {
		securityGroups = append(securityGroups, aws.ToString(sg.VpcSecurityGroupId))
	}
	resource.SetMetadata("security_groups", securityGroups)
	
	// Tags
	if len(cluster.TagList) > 0 {
		for _, tag := range cluster.TagList {
			resource.SetTag(aws.ToString(tag.Key), aws.ToString(tag.Value))
		}
	}
	
	return resource
}

// mapDBStatusToHealth maps RDS status to resource health
func (s *RDSService) mapDBStatusToHealth(status string) string {
	switch strings.ToLower(status) {
	case "available":
		return string(models.HealthHealthy)
	case "creating", "starting", "rebooting", "upgrading", "configuring-enhanced-monitoring":
		return string(models.HealthWarning)
	case "stopped", "stopping", "failed", "storage-full", "incompatible-network", "incompatible-restore":
		return string(models.HealthUnhealthy)
	default:
		return string(models.HealthUnknown)
	}
}

// matchesFilters checks if a resource matches the given filters
func (s *RDSService) matchesFilters(resource *models.Resource, filters shared.ResourceFilters) bool {
	// Check resource type filter
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if strings.EqualFold(rt, "rds") || 
			   strings.EqualFold(rt, "rds_instance") || 
			   strings.EqualFold(rt, "rds_cluster") ||
			   strings.EqualFold(rt, "database") ||
			   strings.EqualFold(rt, "postgres") ||
			   strings.EqualFold(rt, "postgresql") ||
			   strings.EqualFold(rt, "mysql") {
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
	
	// Check creation time filters
	if filters.CreatedAfter != nil && resource.CreatedAt.Before(*filters.CreatedAfter) {
		return false
	}
	
	if filters.CreatedBefore != nil && resource.CreatedAt.After(*filters.CreatedBefore) {
		return false
	}
	
	return true
}

// getRegionsToQuery determines which regions to query based on filters and config
func (s *RDSService) getRegionsToQuery(filterRegions []string) []string {
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

// createRegionClient creates an RDS client for a specific region
func (s *RDSService) createRegionClient(region string) *rds.Client {
	// Create a new config with the specific region
	cfg := s.client.Options()
	cfg.Region = region
	
	return rds.New(cfg)
}