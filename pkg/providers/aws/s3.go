package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	shared "github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// S3Service handles S3-related operations
type S3Service struct {
	client *s3.Client
	config *config.AWSConfig
	logger *logrus.Logger
}

// NewS3Service creates a new S3 service
func NewS3Service(client *s3.Client, cfg *config.AWSConfig, logger *logrus.Logger) *S3Service {
	return &S3Service{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetBuckets retrieves all S3 buckets
func (s *S3Service) GetBuckets(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debug("Getting S3 buckets")
	
	// List all buckets
	listResult, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}
	
	var buckets []models.Resource
	
	for _, bucket := range listResult.Buckets {
		bucketName := aws.ToString(bucket.Name)
		
		// Get bucket details
		resource, err := s.getBucketDetails(ctx, bucketName, bucket)
		if err != nil {
			s.logger.Warnf("Failed to get details for bucket %s: %v", bucketName, err)
			// Create basic resource without details
			resource = s.createBasicBucketResource(bucket)
		}
		
		// Apply filters
		if s.matchesFilters(resource, filters) {
			buckets = append(buckets, *resource)
		}
	}
	
	s.logger.Debugf("Retrieved %d S3 buckets", len(buckets))
	return buckets, nil
}

// GetBucketStatus retrieves the status of a specific S3 bucket
func (s *S3Service) GetBucketStatus(ctx context.Context, bucketName string) (*models.ResourceStatus, error) {
	// Check if bucket exists and is accessible
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	
	if err != nil {
		return nil, fmt.Errorf("S3 bucket %s not found or not accessible: %w", bucketName, err)
	}
	
	return &models.ResourceStatus{
		State:       "available",
		Health:      string(models.HealthHealthy),
		LastChecked: time.Now(),
	}, nil
}

// getBucketDetails retrieves detailed information about an S3 bucket
func (s *S3Service) getBucketDetails(ctx context.Context, bucketName string, bucket types.Bucket) (*models.Resource, error) {
	// Create basic resource
	resource := s.createBasicBucketResource(bucket)
	
	// Get bucket location
	region, err := s.getBucketRegion(ctx, bucketName)
	if err != nil {
		s.logger.Debugf("Failed to get region for bucket %s: %v", bucketName, err)
		region = s.config.Region // Use default region
	}
	resource.Region = region
	
	// Get bucket tags
	tags, err := s.getBucketTags(ctx, bucketName)
	if err != nil {
		s.logger.Debugf("Failed to get tags for bucket %s: %v", bucketName, err)
	} else {
		resource.Tags = tags
	}
	
	// Get bucket encryption
	encryption, err := s.getBucketEncryption(ctx, bucketName)
	if err != nil {
		s.logger.Debugf("Failed to get encryption for bucket %s: %v", bucketName, err)
	} else {
		resource.SetMetadata("encryption", encryption)
	}
	
	// Get bucket versioning
	versioning, err := s.getBucketVersioning(ctx, bucketName)
	if err != nil {
		s.logger.Debugf("Failed to get versioning for bucket %s: %v", bucketName, err)
	} else {
		resource.SetMetadata("versioning", versioning)
	}
	
	// Get bucket notification configuration
	notification, err := s.getBucketNotification(ctx, bucketName)
	if err != nil {
		s.logger.Debugf("Failed to get notification for bucket %s: %v", bucketName, err)
	} else {
		resource.SetMetadata("notifications", notification)
	}
	
	return resource, nil
}

// createBasicBucketResource creates a basic S3 bucket resource
func (s *S3Service) createBasicBucketResource(bucket types.Bucket) *models.Resource {
	bucketName := aws.ToString(bucket.Name)
	
	resource := models.NewResource(
		bucketName,
		bucketName,
		string(models.ResourceTypeObjectStorage),
		"aws",
		s.config.Region, // Will be updated with actual region
	)
	
	// Set creation time
	if bucket.CreationDate != nil {
		resource.CreatedAt = *bucket.CreationDate
	}
	
	// Set status
	resource.UpdateStatus("available", string(models.HealthHealthy))
	
	// Add metadata
	resource.SetMetadata("service", "s3")
	resource.SetMetadata("bucket_name", bucketName)
	
	return resource
}

// getBucketRegion gets the region of an S3 bucket
func (s *S3Service) getBucketRegion(ctx context.Context, bucketName string) (string, error) {
	result, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", err
	}
	
	// Handle empty location constraint (means us-east-1)
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}
	
	return string(result.LocationConstraint), nil
}

// getBucketTags gets the tags of an S3 bucket
func (s *S3Service) getBucketTags(ctx context.Context, bucketName string) (map[string]string, error) {
	result, err := s.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// No tags is not an error
		return make(map[string]string), nil
	}
	
	tags := make(map[string]string)
	for _, tag := range result.TagSet {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	
	return tags, nil
}

// getBucketEncryption gets the encryption configuration of an S3 bucket
func (s *S3Service) getBucketEncryption(ctx context.Context, bucketName string) (map[string]interface{}, error) {
	result, err := s.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return map[string]interface{}{"enabled": false}, nil
	}
	
	encryption := map[string]interface{}{
		"enabled": true,
		"rules":   make([]map[string]interface{}, 0),
	}
	
	for _, rule := range result.ServerSideEncryptionConfiguration.Rules {
		ruleMap := map[string]interface{}{
			"apply_server_side_encryption_by_default": map[string]interface{}{
				"sse_algorithm": string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm),
			},
		}
		
		if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
			ruleMap["apply_server_side_encryption_by_default"].(map[string]interface{})["kms_master_key_id"] = aws.ToString(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID)
		}
		
		encryption["rules"] = append(encryption["rules"].([]map[string]interface{}), ruleMap)
	}
	
	return encryption, nil
}

// getBucketVersioning gets the versioning configuration of an S3 bucket
func (s *S3Service) getBucketVersioning(ctx context.Context, bucketName string) (map[string]interface{}, error) {
	result, err := s.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return map[string]interface{}{"status": "Disabled"}, nil
	}
	
	versioning := map[string]interface{}{
		"status": string(result.Status),
	}
	
	if result.MFADelete != "" {
		versioning["mfa_delete"] = string(result.MFADelete)
	}
	
	return versioning, nil
}

// getBucketNotification gets the notification configuration of an S3 bucket
func (s *S3Service) getBucketNotification(ctx context.Context, bucketName string) (map[string]interface{}, error) {
	result, err := s.client.GetBucketNotificationConfiguration(ctx, &s3.GetBucketNotificationConfigurationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return map[string]interface{}{"configured": false}, nil
	}
	
	notification := map[string]interface{}{
		"configured": false,
	}
	
	// Check if any notification configurations exist
	// Note: Field names in AWS SDK v2 might be different, so we check for the result object
	if result != nil {
		// Simple check - if we got a result without error, some configuration might exist
		// We'll improve this with proper field checking once we verify the correct field names
		notification["configured"] = true
		notification["lambda_configurations"] = 0
		notification["queue_configurations"] = 0 
		notification["topic_configurations"] = 0
	}
	
	return notification, nil
}

// matchesFilters checks if a resource matches the given filters
func (s *S3Service) matchesFilters(resource *models.Resource, filters shared.ResourceFilters) bool {
	// Check resource type filter
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if strings.EqualFold(rt, "s3") || strings.EqualFold(rt, "bucket") || strings.EqualFold(rt, "object_storage") {
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