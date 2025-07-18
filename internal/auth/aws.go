package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsconfig "github.com/Tsahi-Elkayam/cloudview/pkg/config"
)

// AWSAuthenticator handles AWS authentication
type AWSAuthenticator struct {
	config *awsconfig.AWSConfig
	awsCfg aws.Config
}

// NewAWSAuthenticator creates a new AWS authenticator
func NewAWSAuthenticator(cfg *awsconfig.AWSConfig) *AWSAuthenticator {
	return &AWSAuthenticator{
		config: cfg,
	}
}

// Authenticate authenticates with AWS and returns the AWS config
func (a *AWSAuthenticator) Authenticate(ctx context.Context) (aws.Config, error) {
	var cfg aws.Config
	var err error
	
	// Load configuration based on the authentication method
	if a.config.AccessKeyID != "" && a.config.SecretAccessKey != "" {
		// Use static credentials
		cfg, err = a.authenticateWithStaticCredentials(ctx)
	} else if a.config.Profile != "" {
		// Use profile
		cfg, err = a.authenticateWithProfile(ctx)
	} else {
		// Use default credential chain
		cfg, err = a.authenticateWithDefault(ctx)
	}
	
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to authenticate with AWS: %w", err)
	}
	
	// Handle role assumption if specified
	if a.config.RoleARN != "" {
		cfg, err = a.assumeRole(ctx, cfg)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to assume role: %w", err)
		}
	}
	
	a.awsCfg = cfg
	return cfg, nil
}

// authenticateWithStaticCredentials authenticates using static credentials
func (a *AWSAuthenticator) authenticateWithStaticCredentials(ctx context.Context) (aws.Config, error) {
	creds := credentials.NewStaticCredentialsProvider(
		a.config.AccessKeyID,
		a.config.SecretAccessKey,
		a.config.SessionToken,
	)
	
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(a.config.Region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config with static credentials: %w", err)
	}
	
	return cfg, nil
}

// authenticateWithProfile authenticates using AWS profile
func (a *AWSAuthenticator) authenticateWithProfile(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(a.config.Region),
		config.WithSharedConfigProfile(a.config.Profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config with profile %s: %w", a.config.Profile, err)
	}
	
	return cfg, nil
}

// authenticateWithDefault authenticates using default credential chain
func (a *AWSAuthenticator) authenticateWithDefault(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(a.config.Region),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load default AWS config: %w", err)
	}
	
	return cfg, nil
}

// assumeRole assumes an IAM role
func (a *AWSAuthenticator) assumeRole(ctx context.Context, cfg aws.Config) (aws.Config, error) {
	stsClient := sts.NewFromConfig(cfg)
	
	// Create role credentials provider
	roleProvider := stscreds.NewAssumeRoleProvider(stsClient, a.config.RoleARN, func(options *stscreds.AssumeRoleOptions) {
		if a.config.ExternalID != "" {
			options.ExternalID = aws.String(a.config.ExternalID)
		}
		if a.config.MFASerial != "" {
			options.SerialNumber = aws.String(a.config.MFASerial)
		}
		if a.config.DurationSeconds > 0 {
			// Convert int32 seconds to time.Duration
			duration := time.Duration(a.config.DurationSeconds) * time.Second
			options.Duration = duration
		}
		options.RoleSessionName = "cloudview-session"
	})
	
	// Create new config with role credentials
	newCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(a.config.Region),
		config.WithCredentialsProvider(roleProvider),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to assume role %s: %w", a.config.RoleARN, err)
	}
	
	return newCfg, nil
}

// ValidateCredentials validates the AWS credentials by making a test call
func (a *AWSAuthenticator) ValidateCredentials(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	if a.awsCfg.Credentials == nil {
		return nil, fmt.Errorf("no AWS configuration available, call Authenticate first")
	}
	
	stsClient := sts.NewFromConfig(a.awsCfg)
	
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate AWS credentials: %w", err)
	}
	
	return identity, nil
}

// GetConfig returns the authenticated AWS config
func (a *AWSAuthenticator) GetConfig() aws.Config {
	return a.awsCfg
}

// GetRegion returns the configured region
func (a *AWSAuthenticator) GetRegion() string {
	return a.config.Region
}

// GetAllRegions returns all configured regions
func (a *AWSAuthenticator) GetAllRegions() []string {
	if len(a.config.Regions) > 0 {
		return a.config.Regions
	}
	return []string{a.config.Region}
}