package cloudview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
)

// NewConfigCommand creates the config management command
func NewConfigCommand(logger *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CloudView configuration",
		Long: `Manage CloudView configuration files and settings.

CloudView uses smart built-in defaults that work out of the box. You only need to 
create a config file if you want to override specific settings.

Configuration priority (highest to lowest):
  1. Command line flags
  2. Environment variables (CLOUDVIEW_* or AWS_*)
  3. Configuration file (~/.cloudview.yaml)
  4. Built-in defaults

Examples:
  # Show current effective configuration
  cloudview config show

  # Show where CloudView looks for config files
  cloudview config path

  # Generate an example config file to customize
  cloudview config init

  # Validate your current configuration
  cloudview config validate`,
	}

	cmd.AddCommand(NewConfigShowCommand(logger))
	cmd.AddCommand(NewConfigInitCommand(logger))
	cmd.AddCommand(NewConfigPathCommand(logger))
	cmd.AddCommand(NewConfigValidateCommand(logger))

	return cmd
}

// NewConfigShowCommand shows the current effective configuration
func NewConfigShowCommand(logger *logrus.Logger) *cobra.Command {
	var format string
	var showDefaults bool
	var showSources bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current effective configuration",
		Long: `Show the current effective configuration that CloudView is using.

This displays the final configuration after merging:
- Built-in defaults
- Configuration file overrides (if any)
- Environment variable overrides (if any)

Use --show-sources to see where each setting comes from.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load current configuration
			cfg, err := config.DefaultLoader.LoadConfig("")
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Show configuration sources if requested
			if showSources {
				printConfigSources()
				fmt.Printf("\n")
			}

			// Display configuration based on format
			switch strings.ToLower(format) {
			case "yaml":
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				defer encoder.Close()
				return encoder.Encode(cfg)
			case "json":
				encoder := NewJSONEncoder(os.Stdout)
				return encoder.Encode(cfg)
			default:
				return showConfigTable(cfg, showDefaults)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, yaml, json)")
	cmd.Flags().BoolVar(&showDefaults, "show-defaults", false, "Show all settings including defaults")
	cmd.Flags().BoolVar(&showSources, "show-sources", false, "Show where configuration values come from")

	return cmd
}

// NewConfigInitCommand creates a new configuration file
func NewConfigInitCommand(logger *logrus.Logger) *cobra.Command {
	var configFile string
	var force bool
	var minimal bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate an example configuration file",
		Long: `Generate an example configuration file with common settings.

This creates a configuration file with examples of the most commonly overridden 
settings. All settings are optional - CloudView uses sensible defaults for 
anything not specified.

The generated file includes:
- Detailed comments explaining each option
- Examples of different authentication methods
- Common configuration scenarios
- Environment variable alternatives`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine config file path
			if configFile == "" {
				configFile = config.DefaultLoader.GetConfigPath()
			}

			// Check if file exists
			if !force && fileExists(configFile) {
				fmt.Printf("⚠️  Config file already exists: %s\n", configFile)
				fmt.Printf("Use --force to overwrite, or specify a different path with --file\n")
				return nil
			}

			// Generate config file
			if err := config.DefaultLoader.GenerateExampleConfig(configFile); err != nil {
				return fmt.Errorf("failed to generate config file: %w", err)
			}

			fmt.Printf("✅ Generated example configuration file: %s\n\n", configFile)
			
			fmt.Printf("🎯 NEXT STEPS:\n")
			fmt.Printf("   1. Edit the file to customize your settings\n")
			fmt.Printf("   2. Uncomment and modify only the settings you want to change\n")
			fmt.Printf("   3. CloudView will use built-in defaults for everything else\n\n")
			
			fmt.Printf("💡 TIPS:\n")
			fmt.Printf("   • Start with just the AWS profile and regions you use\n")
			fmt.Printf("   • You can delete sections you don't want to customize\n")
			fmt.Printf("   • Use 'cloudview config show' to see your effective configuration\n")
			fmt.Printf("   • Use 'cloudview config validate' to check for errors\n\n")
			
			fmt.Printf("📖 QUICK EDIT:\n")
			fmt.Printf("   vim %s\n", configFile)
			fmt.Printf("   code %s\n", configFile)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "file", "f", "", "Config file path (default: ~/.cloudview.yaml)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config file")
	cmd.Flags().BoolVar(&minimal, "minimal", false, "Generate minimal config with only essential settings")

	return cmd
}

// NewConfigPathCommand shows configuration file paths and search locations
func NewConfigPathCommand(logger *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file locations",
		Long:  `Show where CloudView looks for configuration files and which ones exist.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("🗂️  CloudView Configuration Paths\n")
			fmt.Printf("================================\n\n")
			
			defaultPath := config.DefaultLoader.GetConfigPath()
			fmt.Printf("📝 Default config path: %s\n\n", defaultPath)

			// Check all search locations
			homeDir, _ := os.UserHomeDir()
			locations := []struct {
				path        string
				description string
			}{
				{".cloudview.yaml", "Current directory"},
				{".cloudview.yml", "Current directory (alternative)"},
				{filepath.Join(homeDir, ".cloudview.yaml"), "Home directory"},
				{filepath.Join(homeDir, ".cloudview.yml"), "Home directory (alternative)"},
				{"/etc/cloudview/.cloudview.yaml", "System-wide configuration"},
			}

			fmt.Printf("🔍 Search locations (in order of priority):\n")
			foundAny := false
			for i, location := range locations {
				exists := fileExists(location.path)
				status := "❌ not found"
				if exists {
					status = "✅ found"
					foundAny = true
				}
				
				fmt.Printf("   %d. %s (%s)\n", i+1, location.path, location.description)
				fmt.Printf("      %s\n", status)
			}

			fmt.Printf("\n")
			if !foundAny {
				fmt.Printf("💡 No config file found - CloudView is using built-in defaults.\n")
				fmt.Printf("   This is perfectly fine! CloudView works great with defaults.\n")
				fmt.Printf("   Run 'cloudview config init' to create a config file if you want to customize settings.\n")
			} else {
				fmt.Printf("✅ Found configuration file(s).\n")
				fmt.Printf("   CloudView will use the first one found in the priority order above.\n")
			}

			// Show environment variables that can override config
			fmt.Printf("\n🔧 Environment variables that override config:\n")
			envVars := []struct {
				name        string
				description string
			}{
				{"AWS_PROFILE", "AWS profile to use"},
				{"AWS_REGION", "AWS default region"},
				{"CLOUDVIEW_AWS_PROFILE", "Override AWS profile"},
				{"CLOUDVIEW_AWS_REGION", "Override AWS region"},
				{"CLOUDVIEW_OUTPUT_FORMAT", "Override output format (table/json/yaml)"},
				{"CLOUDVIEW_LOG_LEVEL", "Override log level (debug/info/warn/error)"},
			}

			for _, env := range envVars {
				value := os.Getenv(env.name)
				status := "not set"
				if value != "" {
					status = fmt.Sprintf("= %s", value)
				}
				fmt.Printf("   %s (%s)\n", env.name, status)
			}

			return nil
		},
	}

	return cmd
}

// NewConfigValidateCommand validates the configuration
func NewConfigValidateCommand(logger *logrus.Logger) *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  `Validate the current configuration for errors and warnings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.DefaultLoader.LoadConfig(configFile)
			if err != nil {
				fmt.Printf("❌ Configuration validation failed:\n")
				fmt.Printf("   %v\n", err)
				return err
			}

			fmt.Printf("✅ Configuration is valid!\n\n")

			// Show provider status
			fmt.Printf("🔌 Provider Status:\n")
			for name, providerConfig := range cfg.Providers {
				if providerConfig.IsEnabled() {
					regions := providerConfig.GetRegions()
					fmt.Printf("   ✅ %s: enabled (%d regions: %v)\n", name, len(regions), regions)
				} else {
					fmt.Printf("   ⚪ %s: disabled\n", name)
				}
			}

			// Show configuration summary
			fmt.Printf("\n⚙️  Configuration Summary:\n")
			fmt.Printf("   📊 Output format: %s\n", cfg.Output.Format)
			fmt.Printf("   🎨 Colors: %v\n", cfg.Output.Colors)
			fmt.Printf("   💾 Cache: %v (%s, %v TTL)\n", cfg.Cache.Enabled, cfg.Cache.Storage, cfg.Cache.TTL)
			fmt.Printf("   📝 Logging: %s level, %s format\n", cfg.Logging.Level, cfg.Logging.Format)

			// Show warnings
			warnings := validateConfigWarnings(cfg)
			if len(warnings) > 0 {
				fmt.Printf("\n⚠️  Warnings:\n")
				for _, warning := range warnings {
					fmt.Printf("   • %s\n", warning)
				}
			}

			// Show recommendations  
			recommendations := getConfigRecommendations(cfg)
			if len(recommendations) > 0 {
				fmt.Printf("\n💡 Recommendations:\n")
				for _, rec := range recommendations {
					fmt.Printf("   • %s\n", rec)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "file", "f", "", "Config file to validate (default: auto-detect)")

	return cmd
}

// showConfigTable displays configuration in a readable table format
func showConfigTable(cfg *config.Config, showDefaults bool) error {
	fmt.Printf("🌥️  CloudView Configuration\n")
	fmt.Printf("===========================\n\n")

	// Providers section
	fmt.Printf("🔌 Providers:\n")
	hasEnabledProvider := false
	for name, providerConfig := range cfg.Providers {
		status := "⚪ disabled"
		if providerConfig.IsEnabled() {
			status = "✅ enabled"
			hasEnabledProvider = true
		}
		fmt.Printf("   %s: %s\n", name, status)

		if showDefaults || providerConfig.IsEnabled() {
			if awsConfig, ok := providerConfig.(*config.AWSConfig); ok {
				fmt.Printf("      Profile: %s\n", awsConfig.Profile)
				fmt.Printf("      Region: %s\n", awsConfig.Region)
				if len(awsConfig.Regions) > 0 {
					fmt.Printf("      Regions: %v\n", awsConfig.Regions)
				}
				if awsConfig.RoleARN != "" {
					fmt.Printf("      Role ARN: %s\n", awsConfig.RoleARN)
				}
			}
		}
	}
	
	if !hasEnabledProvider {
		fmt.Printf("   ⚠️  No providers are enabled\n")
	}
	fmt.Printf("\n")

	// Cache section
	fmt.Printf("💾 Cache:\n")
	fmt.Printf("   Enabled: %v\n", cfg.Cache.Enabled)
	if showDefaults || cfg.Cache.Enabled {
		fmt.Printf("   TTL: %v\n", cfg.Cache.TTL)
		fmt.Printf("   Storage: %s\n", cfg.Cache.Storage)
		fmt.Printf("   Max Size: %s\n", cfg.Cache.MaxSize)
	}
	fmt.Printf("\n")

	// Output section
	fmt.Printf("📊 Output:\n")
	fmt.Printf("   Format: %s\n", cfg.Output.Format)
	fmt.Printf("   Colors: %v\n", cfg.Output.Colors)
	if showDefaults || cfg.Output.MaxWidth > 0 {
		fmt.Printf("   Max Width: %d\n", cfg.Output.MaxWidth)
	}
	fmt.Printf("\n")

	// Logging section
	fmt.Printf("📝 Logging:\n")
	fmt.Printf("   Level: %s\n", cfg.Logging.Level)
	fmt.Printf("   Format: %s\n", cfg.Logging.Format)
	fmt.Printf("   Color: %v\n", cfg.Logging.Color)
	if cfg.Logging.File != "" {
		fmt.Printf("   File: %s\n", cfg.Logging.File)
	}

	return nil
}

// printConfigSources shows where configuration values come from
func printConfigSources() {
	fmt.Printf("📋 Configuration Sources:\n")
	
	source := config.DefaultLoader.GetEffectiveConfigSource()
	
	// Config file
	if hasConfigFile, ok := source["config_file"].(bool); ok {
		if hasConfigFile {
			if configPath, ok := source["config_path"].(string); ok {
				fmt.Printf("   📄 Config file: %s (found)\n", configPath)
			}
		} else {
			fmt.Printf("   📄 Config file: none (using defaults)\n")
		}
	}
	
	// Environment variables
	if hasEnvVars, ok := source["env_vars"].(bool); ok {
		if hasEnvVars {
			if setVars, ok := source["set_env_vars"].([]string); ok {
				fmt.Printf("   🔧 Environment variables: %v\n", setVars)
			}
		} else {
			fmt.Printf("   🔧 Environment variables: none set\n")
		}
	}
	
	fmt.Printf("   🏗️  Built-in defaults: always active as base\n")
}

// validateConfigWarnings returns configuration warnings
func validateConfigWarnings(cfg *config.Config) []string {
	var warnings []string

	// Check if any providers are enabled
	if !cfg.HasEnabledProviders() {
		warnings = append(warnings, "No cloud providers are enabled - you won't be able to query any resources")
	}

	// Check for AWS specific warnings
	if awsConfig, ok := cfg.Providers["aws"].(*config.AWSConfig); ok && awsConfig.IsEnabled() {
		if awsConfig.AccessKeyID != "" && awsConfig.SecretAccessKey != "" {
			warnings = append(warnings, "Static AWS credentials found in config - consider using AWS profiles or IAM roles for better security")
		}
		
		if len(awsConfig.Regions) == 1 {
			warnings = append(warnings, "Only one AWS region configured - you might miss resources in other regions")
		}
	}

	return warnings
}

// getConfigRecommendations returns configuration recommendations
func getConfigRecommendations(cfg *config.Config) []string {
	var recommendations []string

	// AWS recommendations
	if awsConfig, ok := cfg.Providers["aws"].(*config.AWSConfig); ok && awsConfig.IsEnabled() {
		if awsConfig.Profile == "default" {
			recommendations = append(recommendations, "Consider using a specific AWS profile name instead of 'default' for better clarity")
		}
		
		if len(awsConfig.Regions) < 3 {
			recommendations = append(recommendations, "Consider adding more AWS regions to get complete resource visibility")
		}
	}

	// Cache recommendations
	if cfg.Cache.Enabled && cfg.Cache.TTL < time.Minute {
		recommendations = append(recommendations, "Cache TTL is very short - consider increasing it for better performance")
	}

	return recommendations
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}