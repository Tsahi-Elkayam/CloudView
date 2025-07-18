package config

import (
	"fmt"
	"time"
)

// Config represents the main application configuration
type Config struct {
	Providers map[string]ProviderConfig `yaml:"providers" json:"providers"`
	Cache     CacheConfig              `yaml:"cache" json:"cache"`
	Output    OutputConfig             `yaml:"output" json:"output"`
	Logging   LoggingConfig            `yaml:"logging" json:"logging"`
}

// ProviderConfig is the interface for all provider configurations
type ProviderConfig interface {
	GetProvider() string
	GetName() string
	IsEnabled() bool
	GetRegions() []string
	Validate() error
}

// BaseProviderConfig contains common provider configuration fields
type BaseProviderConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Regions []string `yaml:"regions" json:"regions"`
}

// GetRegions returns the configured regions
func (c *BaseProviderConfig) GetRegions() []string {
	return c.Regions
}

// IsEnabled returns whether the provider is enabled
func (c *BaseProviderConfig) IsEnabled() bool {
	return c.Enabled
}

// AWSConfig represents AWS provider configuration
type AWSConfig struct {
	BaseProviderConfig `yaml:",inline"`
	Profile            string `yaml:"profile" json:"profile"`
	Region             string `yaml:"region" json:"region"`
	AccessKeyID        string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey    string `yaml:"secret_access_key" json:"secret_access_key"`
	SessionToken       string `yaml:"session_token" json:"session_token"`
	RoleARN            string `yaml:"role_arn" json:"role_arn"`
	ExternalID         string `yaml:"external_id" json:"external_id"`
	MFASerial          string `yaml:"mfa_serial" json:"mfa_serial"`
	DurationSeconds    int32  `yaml:"duration_seconds" json:"duration_seconds"`
}

// GetProvider returns the provider name
func (c *AWSConfig) GetProvider() string {
	return "aws"
}

// GetName returns the provider name
func (c *AWSConfig) GetName() string {
	return "aws"
}

// Validate validates the AWS configuration
func (c *AWSConfig) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if disabled
	}
	
	// At least one authentication method must be provided
	hasCredentials := c.AccessKeyID != "" && c.SecretAccessKey != ""
	hasProfile := c.Profile != ""
	hasRole := c.RoleARN != ""
	
	if !hasCredentials && !hasProfile && !hasRole {
		// In this case, we'll rely on default AWS credential chain
		// This is actually valid - AWS SDK will try instance metadata, etc.
	}
	
	// Validate region
	if c.Region == "" && len(c.Regions) == 0 {
		return fmt.Errorf("AWS provider requires at least one region to be specified")
	}
	
	// Set default region if not specified
	if c.Region == "" && len(c.Regions) > 0 {
		c.Region = c.Regions[0]
	}
	
	// Add region to regions list if not present and regions is empty
	if c.Region != "" && len(c.Regions) == 0 {
		c.Regions = []string{c.Region}
	}
	
	// Validate role assumption parameters
	if c.RoleARN != "" {
		if c.DurationSeconds <= 0 {
			c.DurationSeconds = 3600 // Default 1 hour
		}
		if c.DurationSeconds < 900 || c.DurationSeconds > 43200 {
			return fmt.Errorf("duration_seconds must be between 900 and 43200 seconds")
		}
	}
	
	return nil
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	TTL       time.Duration `yaml:"ttl" json:"ttl"`
	Storage   string        `yaml:"storage" json:"storage"` // memory, disk
	MaxSize   string        `yaml:"max_size" json:"max_size"`
	Directory string        `yaml:"directory" json:"directory"`
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Format   string `yaml:"format" json:"format"`     // table, json, yaml, excel
	Colors   bool   `yaml:"colors" json:"colors"`
	MaxWidth int    `yaml:"max_width" json:"max_width"`
	NoHeader bool   `yaml:"no_header" json:"no_header"`
	Compact  bool   `yaml:"compact" json:"compact"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // trace, debug, info, warn, error, fatal, panic
	Format string `yaml:"format" json:"format"` // text, json
	Color  bool   `yaml:"color" json:"color"`
	File   string `yaml:"file" json:"file"`
}

// DefaultConfig returns a comprehensive default configuration that works out of the box
func DefaultConfig() *Config {
	return &Config{
		Providers: map[string]ProviderConfig{
			"aws": &AWSConfig{
				BaseProviderConfig: BaseProviderConfig{
					Enabled: true, // Enable by default - credentials will be validated at runtime
					Regions: []string{
						"us-east-1", // Primary region (most services, cheapest)
						"us-west-2", // Secondary region (good for multi-region setups)
					},
				},
				Profile:         "default", // Use default AWS profile
				Region:          "us-east-1",
				DurationSeconds: 3600, // 1 hour default for role assumption
			},
		},
		Cache: CacheConfig{
			Enabled:   true,
			TTL:       5 * time.Minute, // 5 minutes - good balance between freshness and performance
			Storage:   "memory",        // Memory is faster and simpler for most use cases
			MaxSize:   "100MB",         // Reasonable default for memory usage
			Directory: "",              // Will be set to temp dir if disk storage is used
		},
		Output: OutputConfig{
			Format:   "table", // Table format is most readable for CLI usage
			Colors:   true,    // Colors make output more readable
			MaxWidth: 0,       // Auto-width based on terminal
			NoHeader: false,   // Headers are helpful
			Compact:  false,   // Full format is more readable
		},
		Logging: LoggingConfig{
			Level:  "info",  // Info level shows important messages without being too verbose
			Format: "text",  // Text format is more readable for CLI usage
			Color:  true,    // Colored logs are easier to read
			File:   "",      // Log to stdout by default
		},
	}
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Validate each provider
	for name, providerConfig := range c.Providers {
		if providerConfig.IsEnabled() {
			if err := providerConfig.Validate(); err != nil {
				return fmt.Errorf("invalid configuration for provider %s: %w", name, err)
			}
		}
	}
	
	// Validate cache config
	if c.Cache.Storage != "memory" && c.Cache.Storage != "disk" {
		return fmt.Errorf("cache storage must be 'memory' or 'disk'")
	}
	
	// Validate output config
	validFormats := []string{"table", "json", "yaml", "excel"}
	validFormat := false
	for _, format := range validFormats {
		if c.Output.Format == format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("output format must be one of: %v", validFormats)
	}
	
	// Validate logging config
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	validLevel := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("logging level must be one of: %v", validLevels)
	}
	
	return nil
}

// GetEnabledProviders returns a map of enabled providers
func (c *Config) GetEnabledProviders() map[string]ProviderConfig {
	enabled := make(map[string]ProviderConfig)
	for name, config := range c.Providers {
		if config.IsEnabled() {
			enabled[name] = config
		}
	}
	return enabled
}

// HasEnabledProviders returns true if at least one provider is enabled
func (c *Config) HasEnabledProviders() bool {
	for _, config := range c.Providers {
		if config.IsEnabled() {
			return true
		}
	}
	return false
}

// GetSummary returns a human-readable summary of the configuration
func (c *Config) GetSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	
	// Provider summary
	providerSummary := make(map[string]interface{})
	for name, config := range c.Providers {
		providerSummary[name] = map[string]interface{}{
			"enabled": config.IsEnabled(),
			"regions": len(config.GetRegions()),
		}
	}
	summary["providers"] = providerSummary
	
	// Cache summary
	summary["cache"] = map[string]interface{}{
		"enabled": c.Cache.Enabled,
		"storage": c.Cache.Storage,
		"ttl":     c.Cache.TTL.String(),
	}
	
	// Output summary
	summary["output"] = map[string]interface{}{
		"format": c.Output.Format,
		"colors": c.Output.Colors,
	}
	
	// Logging summary
	summary["logging"] = map[string]interface{}{
		"level":  c.Logging.Level,
		"format": c.Logging.Format,
	}
	
	return summary
}