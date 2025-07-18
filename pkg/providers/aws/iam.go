package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/sirupsen/logrus"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	shared "github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// IAMService handles IAM-related operations
type IAMService struct {
	client *iam.Client
	config *config.AWSConfig
	logger *logrus.Logger
}

// NewIAMService creates a new IAM service
func NewIAMService(client *iam.Client, cfg *config.AWSConfig, logger *logrus.Logger) *IAMService {
	return &IAMService{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetUsers retrieves all IAM users
func (s *IAMService) GetUsers(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debug("Getting IAM users")
	
	var allUsers []models.Resource
	
	// List users
	paginator := iam.NewListUsersPaginator(s.client, &iam.ListUsersInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM users: %w", err)
		}
		
		for _, user := range page.Users {
			resource := s.convertUserToResource(user)
			
			// Get user's attached policies
			policies, err := s.getUserPolicies(ctx, aws.ToString(user.UserName))
			if err != nil {
				s.logger.Warnf("Failed to get policies for user %s: %v", aws.ToString(user.UserName), err)
			} else {
				resource.SetMetadata("attached_policies", policies)
			}
			
			// Get user's groups
			groups, err := s.getUserGroups(ctx, aws.ToString(user.UserName))
			if err != nil {
				s.logger.Warnf("Failed to get groups for user %s: %v", aws.ToString(user.UserName), err)
			} else {
				resource.SetMetadata("groups", groups)
			}
			
			// Get access keys
			accessKeys, err := s.getUserAccessKeys(ctx, aws.ToString(user.UserName))
			if err != nil {
				s.logger.Warnf("Failed to get access keys for user %s: %v", aws.ToString(user.UserName), err)
			} else {
				resource.SetMetadata("access_keys", accessKeys)
			}
			
			if s.matchesFilters(resource, filters) {
				allUsers = append(allUsers, *resource)
			}
		}
	}
	
	s.logger.Debugf("Retrieved %d IAM users", len(allUsers))
	return allUsers, nil
}

// GetRoles retrieves all IAM roles
func (s *IAMService) GetRoles(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debug("Getting IAM roles")
	
	var allRoles []models.Resource
	
	// List roles
	paginator := iam.NewListRolesPaginator(s.client, &iam.ListRolesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM roles: %w", err)
		}
		
		for _, role := range page.Roles {
			resource := s.convertRoleToResource(role)
			
			// Get role's attached policies
			policies, err := s.getRolePolicies(ctx, aws.ToString(role.RoleName))
			if err != nil {
				s.logger.Warnf("Failed to get policies for role %s: %v", aws.ToString(role.RoleName), err)
			} else {
				resource.SetMetadata("attached_policies", policies)
			}
			
			if s.matchesFilters(resource, filters) {
				allRoles = append(allRoles, *resource)
			}
		}
	}
	
	s.logger.Debugf("Retrieved %d IAM roles", len(allRoles))
	return allRoles, nil
}

// GetPolicies retrieves all IAM policies
func (s *IAMService) GetPolicies(ctx context.Context, filters shared.ResourceFilters) ([]models.Resource, error) {
	s.logger.Debug("Getting IAM policies")
	
	var allPolicies []models.Resource
	
	// List customer managed policies
	paginator := iam.NewListPoliciesPaginator(s.client, &iam.ListPoliciesInput{
		Scope: "Local", // Only customer managed policies
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM policies: %w", err)
		}
		
		for _, policy := range page.Policies {
			resource := s.convertPolicyToResource(policy)
			
			if s.matchesFilters(resource, filters) {
				allPolicies = append(allPolicies, *resource)
			}
		}
	}
	
	s.logger.Debugf("Retrieved %d IAM policies", len(allPolicies))
	return allPolicies, nil
}

// convertUserToResource converts an IAM user to a Resource model
func (s *IAMService) convertUserToResource(user types.User) *models.Resource {
	resource := models.NewResource(
		aws.ToString(user.UserName),
		aws.ToString(user.UserName),
		"iam_user",
		"aws",
		"global", // IAM is global
	)
	
	resource.UpdateStatus("active", string(models.HealthHealthy))
	
	if user.CreateDate != nil {
		resource.CreatedAt = *user.CreateDate
	}
	
	// Add metadata
	resource.SetMetadata("arn", aws.ToString(user.Arn))
	resource.SetMetadata("user_id", aws.ToString(user.UserId))
	resource.SetMetadata("path", aws.ToString(user.Path))
	
	if user.PasswordLastUsed != nil {
		resource.SetMetadata("password_last_used", user.PasswordLastUsed.Format(time.RFC3339))
	}
	
	// Add tags if present
	if len(user.Tags) > 0 {
		for _, tag := range user.Tags {
			resource.SetTag(aws.ToString(tag.Key), aws.ToString(tag.Value))
		}
	}
	
	return resource
}

// convertRoleToResource converts an IAM role to a Resource model  
func (s *IAMService) convertRoleToResource(role types.Role) *models.Resource {
	resource := models.NewResource(
		aws.ToString(role.RoleName),
		aws.ToString(role.RoleName),
		"iam_role", 
		"aws",
		"global",
	)
	
	resource.UpdateStatus("active", string(models.HealthHealthy))
	
	if role.CreateDate != nil {
		resource.CreatedAt = *role.CreateDate
	}
	
	// Add metadata
	resource.SetMetadata("arn", aws.ToString(role.Arn))
	resource.SetMetadata("role_id", aws.ToString(role.RoleId))
	resource.SetMetadata("path", aws.ToString(role.Path))
	resource.SetMetadata("description", aws.ToString(role.Description))
	
	if role.MaxSessionDuration != nil {
		resource.SetMetadata("max_session_duration", *role.MaxSessionDuration)
	}
	
	// Add tags if present
	if len(role.Tags) > 0 {
		for _, tag := range role.Tags {
			resource.SetTag(aws.ToString(tag.Key), aws.ToString(tag.Value))
		}
	}
	
	return resource
}

// convertPolicyToResource converts an IAM policy to a Resource model
func (s *IAMService) convertPolicyToResource(policy types.Policy) *models.Resource {
	resource := models.NewResource(
		aws.ToString(policy.PolicyName),
		aws.ToString(policy.PolicyName),
		"iam_policy",
		"aws", 
		"global",
	)
	
	resource.UpdateStatus("active", string(models.HealthHealthy))
	
	if policy.CreateDate != nil {
		resource.CreatedAt = *policy.CreateDate
	}
	
	if policy.UpdateDate != nil {
		resource.UpdatedAt = *policy.UpdateDate
	}
	
	// Add metadata
	resource.SetMetadata("arn", aws.ToString(policy.Arn))
	resource.SetMetadata("policy_id", aws.ToString(policy.PolicyId))
	resource.SetMetadata("path", aws.ToString(policy.Path))
	resource.SetMetadata("description", aws.ToString(policy.Description))
	resource.SetMetadata("attachment_count", policy.AttachmentCount)
	resource.SetMetadata("default_version_id", aws.ToString(policy.DefaultVersionId))
	
	// Add tags if present
	if len(policy.Tags) > 0 {
		for _, tag := range policy.Tags {
			resource.SetTag(aws.ToString(tag.Key), aws.ToString(tag.Value))
		}
	}
	
	return resource
}

// getUserPolicies gets attached policies for a user
func (s *IAMService) getUserPolicies(ctx context.Context, userName string) ([]string, error) {
	var policies []string
	
	// Get attached managed policies
	result, err := s.client.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return policies, err
	}
	
	for _, policy := range result.AttachedPolicies {
		policies = append(policies, aws.ToString(policy.PolicyName))
	}
	
	// Get inline policies
	inlineResult, err := s.client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return policies, err
	}
	
	for _, policyName := range inlineResult.PolicyNames {
		policies = append(policies, policyName+" (inline)")
	}
	
	return policies, nil
}

// getUserGroups gets groups for a user
func (s *IAMService) getUserGroups(ctx context.Context, userName string) ([]string, error) {
	var groups []string
	
	result, err := s.client.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return groups, err
	}
	
	for _, group := range result.Groups {
		groups = append(groups, aws.ToString(group.GroupName))
	}
	
	return groups, nil
}

// getUserAccessKeys gets access keys for a user
func (s *IAMService) getUserAccessKeys(ctx context.Context, userName string) ([]map[string]interface{}, error) {
	var accessKeys []map[string]interface{}
	
	result, err := s.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return accessKeys, err
	}
	
	for _, key := range result.AccessKeyMetadata {
		keyInfo := map[string]interface{}{
			"access_key_id": aws.ToString(key.AccessKeyId),
			"status":        string(key.Status),
			"create_date":   key.CreateDate.Format(time.RFC3339),
		}
		accessKeys = append(accessKeys, keyInfo)
	}
	
	return accessKeys, nil
}

// getRolePolicies gets attached policies for a role
func (s *IAMService) getRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	var policies []string
	
	// Get attached managed policies
	result, err := s.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return policies, err
	}
	
	for _, policy := range result.AttachedPolicies {
		policies = append(policies, aws.ToString(policy.PolicyName))
	}
	
	// Get inline policies
	inlineResult, err := s.client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return policies, err
	}
	
	for _, policyName := range inlineResult.PolicyNames {
		policies = append(policies, policyName+" (inline)")
	}
	
	return policies, nil
}

// matchesFilters checks if a resource matches the given filters
func (s *IAMService) matchesFilters(resource *models.Resource, filters shared.ResourceFilters) bool {
	// Check resource type filter
	if len(filters.ResourceTypes) > 0 {
		found := false
		for _, rt := range filters.ResourceTypes {
			if strings.EqualFold(rt, "iam") || 
			   strings.EqualFold(rt, "iam_user") || 
			   strings.EqualFold(rt, "iam_role") || 
			   strings.EqualFold(rt, "iam_policy") ||
			   strings.EqualFold(rt, "user") ||
			   strings.EqualFold(rt, "role") ||
			   strings.EqualFold(rt, "policy") {
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