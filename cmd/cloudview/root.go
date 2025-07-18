package cloudview

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	
	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
)

var (
	cfgFile string
	verbose bool
	version = "dev" // This will be set during build
	
	// Global configuration instance
	globalConfig *config.Config
)

// NewRootCommand creates the root command for CloudView CLI
func NewRootCommand(logger *logrus.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cloudview",
		Short: "Cloud-agnostic CLI tool for multi-cloud resource management",
		Long: `CloudView is a plugin-based, cloud-agnostic CLI tool that provides 
a unified interface for managing and monitoring resources across multiple cloud providers.

🏗️  Built-in defaults work out of the box - no configuration required!
📝  Create a config file only to override specific settings you want to change
🔧  Environment variables can override any configuration setting

Currently supported providers:
  ✅ AWS (EC2, S3, RDS, IAM, VPC, Security Groups)
  🚧 GCP (planned)  
  🚧 Azure (planned)

Configuration priority (highest to lowest):
  1. Command line flags
  2. Environment variables  
  3. Configuration file (~/.cloudview.yaml)
  4. Built-in defaults`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load configuration
			var err error
			globalConfig, err = config.DefaultLoader.LoadConfig(cfgFile)
			if err != nil {
				logger.Fatalf("Failed to load configuration: %v", err)
			}
			
			// Set log level based on verbose flag or config
			if verbose {
				logger.SetLevel(logrus.DebugLevel)
			} else {
				if level, err := logrus.ParseLevel(globalConfig.Logging.Level); err == nil {
					logger.SetLevel(level)
				}
			}
			
			// Configure logger format and color
			if globalConfig.Logging.Format == "json" {
				logger.SetFormatter(&logrus.JSONFormatter{
					TimestampFormat: "2006-01-02 15:04:05",
				})
			} else {
				logger.SetFormatter(&logrus.TextFormatter{
					FullTimestamp:   true,
					TimestampFormat: "2006-01-02 15:04:05",
					DisableColors:   !globalConfig.Logging.Color,
				})
			}
			
			// Set output file if specified
			if globalConfig.Logging.File != "" {
				if file, err := os.OpenFile(globalConfig.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
					logger.SetOutput(file)
				} else {
					logger.Warnf("Failed to open log file %s: %v", globalConfig.Logging.File, err)
				}
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			printWelcomeMessage()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", 
		"config file (default: searches for .cloudview.yaml in ., ~, /etc/cloudview)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, 
		"verbose output (overrides config log level)")

	// Add subcommands
	rootCmd.AddCommand(NewInventoryCommand(logger))
	rootCmd.AddCommand(NewConfigCommand(logger))

	return rootCmd
}

// printWelcomeMessage prints a helpful welcome message
func printWelcomeMessage() {
	fmt.Printf(`
┌─────────────────────────────────────────┐
│          🌥️  CloudView CLI               │
│     Multi-Cloud Resource Management     │
└─────────────────────────────────────────┘

Welcome to CloudView! 🚀

🔍 QUICK START:
   cloudview inventory                    # Show all resources
   cloudview inventory --provider aws     # Show AWS resources
   cloudview inventory --type ec2         # Show EC2 instances
   cloudview inventory --help             # More inventory options

⚙️  CONFIGURATION:
   cloudview config show                  # View current config
   cloudview config init                  # Create config file
   cloudview config path                  # Show config locations

📖 EXAMPLES:
   # List resources with filters
   cloudview inventory --provider aws --region us-east-1 --type ec2
   
   # Export to different formats
   cloudview inventory --output json
   cloudview inventory --output yaml
   
   # Filter by tags and status
   cloudview inventory --tag Environment=prod --status running

💡 TIPS:
   • No configuration needed to get started - CloudView uses smart defaults
   • Only create a config file if you want to override specific settings
   • Use environment variables for CI/CD: AWS_PROFILE, CLOUDVIEW_OUTPUT_FORMAT, etc.
   • Run commands with --verbose for detailed logging

Version: %s
For help: cloudview --help
`, version)

	// Show configuration status
	if globalConfig != nil {
		fmt.Printf("\n📊 CURRENT STATUS:\n")
		
		enabledProviders := globalConfig.GetEnabledProviders()
		if len(enabledProviders) > 0 {
			fmt.Printf("   ✅ Enabled providers: ")
			names := make([]string, 0, len(enabledProviders))
			for name := range enabledProviders {
				names = append(names, name)
			}
			fmt.Printf("%v\n", names)
		} else {
			fmt.Printf("   ⚠️  No providers enabled\n")
		}
		
		fmt.Printf("   🗂️  Output format: %s\n", globalConfig.Output.Format)
		fmt.Printf("   📝 Log level: %s\n", globalConfig.Logging.Level)
		
		// Show config source
		source := config.DefaultLoader.GetEffectiveConfigSource()
		if hasConfigFile, ok := source["config_file"].(bool); ok && hasConfigFile {
			if configPath, ok := source["config_path"].(string); ok {
				fmt.Printf("   📄 Config file: %s\n", configPath)
			}
		} else {
			fmt.Printf("   📄 Using built-in defaults (no config file)\n")
		}
	}
	
	fmt.Printf("\n")
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *config.Config {
	return globalConfig
}

// JSONEncoder provides JSON encoding
type JSONEncoder struct {
	encoder *json.Encoder
}

// NewJSONEncoder creates a new JSON encoder
func NewJSONEncoder(w io.Writer) *JSONEncoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return &JSONEncoder{encoder: encoder}
}

// Encode encodes the given value as JSON
func (e *JSONEncoder) Encode(v interface{}) error {
	return e.encoder.Encode(v)
}

// YAMLEncoder provides YAML encoding
type YAMLEncoder struct {
	encoder *yaml.Encoder
}

// NewYAMLEncoder creates a new YAML encoder
func NewYAMLEncoder(w io.Writer) *YAMLEncoder {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return &YAMLEncoder{encoder: encoder}
}

// Encode encodes the given value as YAML
func (e *YAMLEncoder) Encode(v interface{}) error {
	return e.encoder.Encode(v)
}