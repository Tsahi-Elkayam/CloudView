package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading from various sources
type Loader struct {
	configPaths []string
	configName  string
	configType  string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	homeDir, _ := os.UserHomeDir()
	return &Loader{
		configPaths: []string{
			".",
			homeDir,
			"/etc/cloudview",
		},
		configName: ".cloudview",
		configType: "yaml",
	}
}

// LoadConfig loads configuration with proper merging of defaults and user config
func (l *Loader) LoadConfig(configFile string) (*Config, error) {
	// Start with default configuration as the base
	config := DefaultConfig()
	
	// Configure viper
	v := viper.New()
	v.SetConfigType(l.configType)
	
	// Set config file if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName(l.configName)
		for _, path := range l.configPaths {
			v.AddConfigPath(path)
		}
	}
	
	// Set environment variable settings
	v.SetEnvPrefix("CLOUDVIEW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	
	// Bind environment variables
	l.bindEnvironmentVariables(v)
	
	// Try to read config file
	configFileExists := false
	configFilePath := ""
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay, we'll use defaults
		fmt.Printf("No config file found, using built-in defaults\n")
	} else {
		configFileExists = true
		configFilePath = v.ConfigFileUsed()
		fmt.Printf("Using config file: %s\n", configFilePath)
	}
	
	// Merge configuration (only if config file exists or env vars are set)
	if configFileExists || l.hasRelevantEnvVars() {
		if err := l.mergeWithDefaults(v, config); err != nil {
			return nil, fmt.Errorf("failed to merge configuration: %w", err)
		}
		
		if configFileExists {
			fmt.Printf("✅ Configuration loaded with user overrides\n")
		}
		
		if l.hasRelevantEnvVars() {
			fmt.Printf("🔧 Environment variable overrides applied\n")
		}
	} else {
		fmt.Printf("🚀 Using built-in defaults - all systems ready!\n")
	}
	
	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// mergeWithDefaults merges user configuration with defaults, preserving defaults unless explicitly overridden
func (l *Loader) mergeWithDefaults(v *viper.Viper, defaultConfig *Config) error {
	// Create a map to hold the loaded config
	var userConfig map[string]interface{}
	if err := v.Unmarshal(&userConfig); err != nil {
		return fmt.Errorf("failed to unmarshal user config: %w", err)
	}
	
	// Merge providers configuration
	if providers, exists := userConfig["providers"]; exists {
		if providerMap, ok := providers.(map[string]interface{}); ok {
			if err := l.mergeProviders(providerMap, defaultConfig); err != nil {
				return fmt.Errorf("failed to merge providers: %w", err)
			}
		}
	}
	
	// Merge cache configuration
	if cache, exists := userConfig["cache"]; exists {
		if err := l.mergeStruct(cache, &defaultConfig.Cache); err != nil {
			return fmt.Errorf("failed to merge cache config: %w", err)
		}
	}
	
	// Merge output configuration
	if output, exists := userConfig["output"]; exists {
		if err := l.mergeStruct(output, &defaultConfig.Output); err != nil {
			return fmt.Errorf("failed to merge output config: %w", err)
		}
	}
	
	// Merge logging configuration
	if logging, exists := userConfig["logging"]; exists {
		if err := l.mergeStruct(logging, &defaultConfig.Logging); err != nil {
			return fmt.Errorf("failed to merge logging config: %w", err)
		}
	}
	
	return nil
}

// mergeProviders merges provider configurations with defaults intelligently
func (l *Loader) mergeProviders(userProviders map[string]interface{}, defaultConfig *Config) error {
	// AWS provider merging
	if awsData, exists := userProviders["aws"]; exists {
		// Get the default AWS config
		defaultAWS, ok := defaultConfig.Providers["aws"].(*AWSConfig)
		if !ok {
			return fmt.Errorf("default AWS config is not of correct type")
		}
		
		// Create a copy of the default config to avoid modifying the original
		mergedAWS := &AWSConfig{
			BaseProviderConfig: BaseProviderConfig{
				Enabled: defaultAWS.Enabled,
				Regions: make([]string, len(defaultAWS.Regions)),
			},
			Profile:         defaultAWS.Profile,
			Region:          defaultAWS.Region,
			AccessKeyID:     defaultAWS.AccessKeyID,
			SecretAccessKey: defaultAWS.SecretAccessKey,
			SessionToken:    defaultAWS.SessionToken,
			RoleARN:         defaultAWS.RoleARN,
			ExternalID:      defaultAWS.ExternalID,
			MFASerial:       defaultAWS.MFASerial,
			DurationSeconds: defaultAWS.DurationSeconds,
		}
		copy(mergedAWS.Regions, defaultAWS.Regions)
		
		// Merge user config into the default copy
		if err := l.mergeStruct(awsData, mergedAWS); err != nil {
			return fmt.Errorf("failed to merge AWS config: %w", err)
		}
		
		// Special handling for regions - if user specifies regions, use those instead of adding to defaults
		if awsMap, ok := awsData.(map[string]interface{}); ok {
			if userRegions, exists := awsMap["regions"]; exists {
				if regionList, ok := userRegions.([]interface{}); ok {
					var regions []string
					for _, region := range regionList {
						if regionStr, ok := region.(string); ok {
							regions = append(regions, regionStr)
						}
					}
					// Replace regions completely if user provides them
					if len(regions) > 0 {
						mergedAWS.Regions = regions
					}
				}
			}
		}
		
		defaultConfig.Providers["aws"] = mergedAWS
	}
	
	// Future providers (GCP, Azure) would be handled here similarly
	
	return nil
}

// mergeStruct merges data into a target struct, preserving existing values unless explicitly overridden
func (l *Loader) mergeStruct(data interface{}, target interface{}) error {
	// Convert data to YAML bytes
	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	// Unmarshal into target, which will override only the fields present in data
	return yaml.Unmarshal(dataBytes, target)
}

// hasRelevantEnvVars checks if any CloudView environment variables are set
func (l *Loader) hasRelevantEnvVars() bool {
	envVars := []string{
		"CLOUDVIEW_AWS_ENABLED",
		"CLOUDVIEW_AWS_PROFILE",
		"CLOUDVIEW_AWS_REGION",
		"CLOUDVIEW_AWS_ACCESS_KEY_ID",
		"CLOUDVIEW_AWS_SECRET_ACCESS_KEY",
		"CLOUDVIEW_CACHE_ENABLED",
		"CLOUDVIEW_OUTPUT_FORMAT",
		"CLOUDVIEW_LOG_LEVEL",
		"AWS_PROFILE",
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	}
	
	for _, envVar := range envVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	
	return false
}

// bindEnvironmentVariables binds environment variables to viper
func (l *Loader) bindEnvironmentVariables(v *viper.Viper) {
	// AWS configuration
	v.BindEnv("providers.aws.enabled", "CLOUDVIEW_AWS_ENABLED")
	v.BindEnv("providers.aws.profile", "CLOUDVIEW_AWS_PROFILE", "AWS_PROFILE")
	v.BindEnv("providers.aws.region", "CLOUDVIEW_AWS_REGION", "AWS_REGION", "AWS_DEFAULT_REGION")
	v.BindEnv("providers.aws.access_key_id", "CLOUDVIEW_AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	v.BindEnv("providers.aws.secret_access_key", "CLOUDVIEW_AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")
	v.BindEnv("providers.aws.session_token", "CLOUDVIEW_AWS_SESSION_TOKEN", "AWS_SESSION_TOKEN")
	v.BindEnv("providers.aws.role_arn", "CLOUDVIEW_AWS_ROLE_ARN")
	v.BindEnv("providers.aws.external_id", "CLOUDVIEW_AWS_EXTERNAL_ID")
	v.BindEnv("providers.aws.mfa_serial", "CLOUDVIEW_AWS_MFA_SERIAL")
	v.BindEnv("providers.aws.duration_seconds", "CLOUDVIEW_AWS_DURATION_SECONDS")
	
	// Cache configuration
	v.BindEnv("cache.enabled", "CLOUDVIEW_CACHE_ENABLED")
	v.BindEnv("cache.ttl", "CLOUDVIEW_CACHE_TTL")
	v.BindEnv("cache.storage", "CLOUDVIEW_CACHE_STORAGE")
	v.BindEnv("cache.max_size", "CLOUDVIEW_CACHE_MAX_SIZE")
	v.BindEnv("cache.directory", "CLOUDVIEW_CACHE_DIRECTORY")
	
	// Output configuration
	v.BindEnv("output.format", "CLOUDVIEW_OUTPUT_FORMAT")
	v.BindEnv("output.colors", "CLOUDVIEW_OUTPUT_COLORS")
	v.BindEnv("output.max_width", "CLOUDVIEW_OUTPUT_MAX_WIDTH")
	v.BindEnv("output.no_header", "CLOUDVIEW_OUTPUT_NO_HEADER")
	v.BindEnv("output.compact", "CLOUDVIEW_OUTPUT_COMPACT")
	
	// Logging configuration
	v.BindEnv("logging.level", "CLOUDVIEW_LOG_LEVEL")
	v.BindEnv("logging.format", "CLOUDVIEW_LOG_FORMAT")
	v.BindEnv("logging.color", "CLOUDVIEW_LOG_COLOR")
	v.BindEnv("logging.file", "CLOUDVIEW_LOG_FILE")
}

// SaveConfig saves configuration to a file
func (l *Loader) SaveConfig(config *Config, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GenerateExampleConfig generates an example configuration file with comments
func (l *Loader) GenerateExampleConfig(filePath string) error {
	// Create YAML content with helpful comments
	yamlContent := `# CloudView Configuration File
# This file overrides the built-in defaults - only specify settings you want to change
# CloudView will use sensible defaults for anything not specified here

providers:
  aws:
    enabled: true
    
    # Authentication (choose one method):
    profile: "default"  # AWS profile to use (recommended)
    
    # Uncomment for static credentials (not recommended for production):
    # access_key_id: "your_access_key"
    # secret_access_key: "your_secret_key"
    # session_token: "optional_session_token"
    
    # Uncomment for role assumption:
    # role_arn: "arn:aws:iam::123456789012:role/CloudViewRole"
    # external_id: "optional_external_id"
    # mfa_serial: "arn:aws:iam::123456789012:mfa/username"
    # duration_seconds: 3600
    
    # Region configuration:
    region: "us-east-1"  # Primary region
    regions:  # Regions to scan for resources
      - "us-east-1"
      - "us-west-2"
      # Add more regions where you have resources

# Optional: Override cache settings
# cache:
#   enabled: true
#   ttl: "5m"
#   storage: "memory"  # or "disk"
#   max_size: "100MB"

# Optional: Override output settings  
# output:
#   format: "table"  # table, json, yaml
#   colors: true
#   max_width: 0  # 0 = auto

# Optional: Override logging settings
# logging:
#   level: "info"  # trace, debug, info, warn, error
#   format: "text"  # text, json
#   color: true
#   # file: "/path/to/logfile"

# Environment Variable Examples:
# Instead of this file, you can use environment variables:
# export CLOUDVIEW_AWS_PROFILE=myprofile
# export CLOUDVIEW_AWS_REGION=us-west-2  
# export CLOUDVIEW_OUTPUT_FORMAT=json
# export CLOUDVIEW_LOG_LEVEL=debug
`
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Write to file
	return os.WriteFile(filePath, []byte(yamlContent), 0644)
}

// GetConfigPath returns the default path to the configuration file
func (l *Loader) GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, l.configName+".yaml")
}

// ConfigExists checks if a configuration file exists
func (l *Loader) ConfigExists(configFile string) bool {
	if configFile != "" {
		_, err := os.Stat(configFile)
		return err == nil
	}
	
	for _, path := range l.configPaths {
		configPath := filepath.Join(path, l.configName+".yaml")
		if _, err := os.Stat(configPath); err == nil {
			return true
		}
		
		configPath = filepath.Join(path, l.configName+".yml")
		if _, err := os.Stat(configPath); err == nil {
			return true
		}
	}
	
	return false
}

// GetEffectiveConfigSource returns information about where configuration is coming from
func (l *Loader) GetEffectiveConfigSource() map[string]interface{} {
	source := make(map[string]interface{})
	
	// Check for config file
	if l.ConfigExists("") {
		source["config_file"] = true
		source["config_path"] = l.GetConfigPath()
	} else {
		source["config_file"] = false
	}
	
	// Check for environment variables
	source["env_vars"] = l.hasRelevantEnvVars()
	
	// Check specific env vars that are set
	setEnvVars := []string{}
	checkVars := []string{
		"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID",
		"CLOUDVIEW_AWS_PROFILE", "CLOUDVIEW_AWS_REGION", "CLOUDVIEW_OUTPUT_FORMAT",
		"CLOUDVIEW_LOG_LEVEL", "CLOUDVIEW_CACHE_ENABLED",
	}
	
	for _, envVar := range checkVars {
		if os.Getenv(envVar) != "" {
			setEnvVars = append(setEnvVars, envVar)
		}
	}
	source["set_env_vars"] = setEnvVars
	
	return source
}

// DefaultLoader is the global configuration loader
var DefaultLoader = NewLoader()