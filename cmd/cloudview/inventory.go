package cloudview

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Tsahi-Elkayam/cloudview/pkg/config"
	"github.com/Tsahi-Elkayam/cloudview/pkg/models"
	"github.com/Tsahi-Elkayam/cloudview/pkg/providers"
	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
)

// InventoryOptions holds options for the inventory command
type InventoryOptions struct {
	Providers     []string
	Regions       []string
	ResourceTypes []string
	Tags          []string
	Status        []string
	Output        string
	CreatedAfter  string
	CreatedBefore string
	NoHeader      bool
	Verbose       bool
	Wide          bool  // New: Wide table format
	MaxWidth      int   // New: Maximum table width
	NoTruncate    bool  // New: Don't truncate long names
}

// NewInventoryCommand creates the inventory command
func NewInventoryCommand(logger *logrus.Logger) *cobra.Command {
	opts := &InventoryOptions{}
	
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Get resource inventory across cloud providers",
		Long: `Get a comprehensive inventory of resources across multiple cloud providers.

The inventory command allows you to discover and list resources from various cloud providers
with flexible filtering options. You can filter by provider, region, resource type, tags,
and other criteria.

Examples:
  # List all resources from AWS
  cloudview inventory --provider aws

  # List EC2 instances in specific regions with wide format
  cloudview inventory --provider aws --type ec2 --region us-east-1,us-west-2 --wide

  # List resources with specific tags (full names, no truncation)
  cloudview inventory --provider aws --tag Environment=prod,Team=backend --no-truncate

  # List resources created in the last 7 days with custom table width
  cloudview inventory --provider aws --created-after 2024-01-01 --max-width 200

  # Export everything to JSON for analysis
  cloudview inventory --provider aws --output json > infrastructure.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInventoryCommand(cmd.Context(), opts, logger)
		},
	}
	
	// Provider options
	cmd.Flags().StringSliceVarP(&opts.Providers, "provider", "p", []string{"all"}, 
		"Cloud providers to query (aws, all)")
	
	// Filtering options
	cmd.Flags().StringSliceVarP(&opts.Regions, "region", "r", []string{}, 
		"Regions to query (comma-separated)")
	cmd.Flags().StringSliceVarP(&opts.ResourceTypes, "type", "t", []string{}, 
		"Resource types to filter (ec2,s3,lambda,etc)")
	cmd.Flags().StringSliceVar(&opts.Tags, "tag", []string{}, 
		"Tags to filter by (format: key=value)")
	cmd.Flags().StringSliceVarP(&opts.Status, "status", "s", []string{}, 
		"Resource status to filter by (running,stopped,etc)")
	
	// Time filtering
	cmd.Flags().StringVar(&opts.CreatedAfter, "created-after", "", 
		"Show resources created after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.CreatedBefore, "created-before", "", 
		"Show resources created before this date (YYYY-MM-DD)")
	
	// Output options
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "table", 
		"Output format (table,json,yaml)")
	cmd.Flags().BoolVar(&opts.NoHeader, "no-header", false, 
		"Don't print column headers")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, 
		"Verbose output")
	
	// Table formatting options
	cmd.Flags().BoolVarP(&opts.Wide, "wide", "w", false,
		"Wide table format with more spacing")
	cmd.Flags().IntVar(&opts.MaxWidth, "max-width", 0,
		"Maximum table width (0 = auto)")
	cmd.Flags().BoolVar(&opts.NoTruncate, "no-truncate", false,
		"Don't truncate long resource names and values")
	
	return cmd
}

// runInventoryCommand executes the inventory command
func runInventoryCommand(ctx context.Context, opts *InventoryOptions, logger *logrus.Logger) error {
	// Use the global configuration loaded in PersistentPreRun
	cfg := GetGlobalConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}
	
	// Parse filters
	filters, err := parseInventoryFilters(opts)
	if err != nil {
		return fmt.Errorf("failed to parse filters: %w", err)
	}
	
	if opts.Verbose {
		logger.SetLevel(logrus.DebugLevel)
		logger.Debugf("Using filters: %+v", filters)
	}
	
	// Validate that at least one provider is enabled and requested
	enabledProviders := cfg.GetEnabledProviders()
	if len(enabledProviders) == 0 {
		fmt.Printf("⚠️  No cloud providers are enabled in configuration.\n")
		fmt.Printf("💡 Run 'cloudview config init' to create a configuration file,\n")
		fmt.Printf("   or set AWS_PROFILE environment variable to use AWS.\n")
		return nil
	}
	
	// Filter requested providers to only enabled ones
	var validProviders []string
	for _, requestedProvider := range opts.Providers {
		if requestedProvider == "all" {
			// Add all enabled providers
			for name := range enabledProviders {
				validProviders = append(validProviders, name)
			}
			break
		} else if _, exists := enabledProviders[requestedProvider]; exists {
			validProviders = append(validProviders, requestedProvider)
		} else {
			logger.Warnf("Provider %s is not enabled or not supported", requestedProvider)
		}
	}
	
	if len(validProviders) == 0 {
		fmt.Printf("⚠️  None of the requested providers are enabled: %v\n", opts.Providers)
		fmt.Printf("💡 Enabled providers: %v\n", getEnabledProviderNames(enabledProviders))
		fmt.Printf("   Use --provider with one of the enabled providers.\n")
		return nil
	}
	
	logger.Debugf("Querying providers: %v", validProviders)
	
	// Create provider factory
	factory := providers.NewProviderFactory(providers.DefaultRegistry, logger)
	
	// Collect resources from all requested providers
	var allResources []models.Resource
	
	for _, providerName := range validProviders {
		logger.Debugf("Querying provider: %s", providerName)
		
		// Get provider configuration
		providerConfig := enabledProviders[providerName]
		
		// Create provider instance
		provider, err := factory.CreateProvider(ctx, providerName, providerConfig)
		if err != nil {
			logger.Errorf("Failed to create provider %s: %v", providerName, err)
			fmt.Printf("❌ Failed to initialize %s provider: %v\n", providerName, err)
			continue
		}
		
		// Get resources from provider
		fmt.Printf("🔍 Querying %s resources...\n", providerName)
		resources, err := provider.GetResources(ctx, filters)
		if err != nil {
			logger.Errorf("Failed to get resources from provider %s: %v", providerName, err)
			fmt.Printf("❌ Failed to get resources from %s: %v\n", providerName, err)
			continue
		}
		
		allResources = append(allResources, resources...)
		logger.Debugf("Retrieved %d resources from provider %s", len(resources), providerName)
		
		if len(resources) > 0 {
			fmt.Printf("✅ Found %d resources from %s\n", len(resources), providerName)
		} else {
			fmt.Printf("ℹ️  No resources found in %s matching the specified criteria\n", providerName)
		}
	}
	
	fmt.Printf("\n")
	
	if len(allResources) == 0 {
		fmt.Printf("🔍 No resources found matching the specified criteria.\n\n")
		fmt.Printf("💡 TIPS:\n")
		fmt.Printf("   • Check if you have resources in the specified regions: %v\n", filters.Regions)
		if len(filters.ResourceTypes) > 0 {
			fmt.Printf("   • Try removing the --type filter to see all resource types\n")
		}
		if len(filters.Tags) > 0 {
			fmt.Printf("   • Try removing the --tag filters to see all resources\n")
		}
		fmt.Printf("   • Run without filters to see all resources: cloudview inventory\n")
		fmt.Printf("   • Use --verbose for detailed logging\n")
		return nil
	}
	
	// Output results
	fmt.Printf("📊 Found %d total resources\n\n", len(allResources))
	return outputInventoryResults(allResources, opts, logger)
}

// getEnabledProviderNames returns a slice of enabled provider names
func getEnabledProviderNames(enabledProviders map[string]config.ProviderConfig) []string {
	names := make([]string, 0, len(enabledProviders))
	for name := range enabledProviders {
		names = append(names, name)
	}
	return names
}

// parseInventoryFilters parses command line options into resource filters
func parseInventoryFilters(opts *InventoryOptions) (types.ResourceFilters, error) {
	filters := types.ResourceFilters{
		Regions:       opts.Regions,
		ResourceTypes: opts.ResourceTypes,
		Status:        opts.Status,
		Tags:          make(map[string]string),
	}
	
	// Parse tags
	for _, tagStr := range opts.Tags {
		parts := strings.SplitN(tagStr, "=", 2)
		if len(parts) != 2 {
			return filters, fmt.Errorf("invalid tag format: %s (expected key=value)", tagStr)
		}
		filters.Tags[parts[0]] = parts[1]
	}
	
	// Parse time filters
	if opts.CreatedAfter != "" {
		t, err := time.Parse("2006-01-02", opts.CreatedAfter)
		if err != nil {
			return filters, fmt.Errorf("invalid created-after date format: %s (expected YYYY-MM-DD)", opts.CreatedAfter)
		}
		filters.CreatedAfter = &t
	}
	
	if opts.CreatedBefore != "" {
		t, err := time.Parse("2006-01-02", opts.CreatedBefore)
		if err != nil {
			return filters, fmt.Errorf("invalid created-before date format: %s (expected YYYY-MM-DD)", opts.CreatedBefore)
		}
		filters.CreatedBefore = &t
	}
	
	return filters, nil
}

// outputInventoryResults outputs the inventory results in the specified format
func outputInventoryResults(resources []models.Resource, opts *InventoryOptions, logger *logrus.Logger) error {
	switch strings.ToLower(opts.Output) {
	case "json":
		return outputInventoryJSON(resources, opts)
	case "yaml":
		return outputInventoryYAML(resources, opts)
	case "table":
		fallthrough
	default:
		return outputInventoryTable(resources, opts)
	}
}

// outputInventoryTable outputs resources in table format
func outputInventoryTable(resources []models.Resource, opts *InventoryOptions) error {
	// Determine column widths based on options
	var idWidth, nameWidth, typeWidth, regionWidth, statusWidth, tagsWidth int
	
	if opts.NoTruncate {
		// Calculate max widths from actual data
		idWidth, nameWidth, typeWidth, regionWidth, statusWidth, tagsWidth = calculateOptimalWidths(resources)
	} else if opts.Wide {
		// Wide format with generous spacing
		idWidth, nameWidth, typeWidth, regionWidth, statusWidth, tagsWidth = 30, 25, 25, 15, 15, 50
	} else {
		// Standard format
		idWidth, nameWidth, typeWidth, regionWidth, statusWidth, tagsWidth = 20, 15, 20, 15, 12, 40
	}
	
	// Apply max width limit if specified
	if opts.MaxWidth > 0 {
		totalWidth := idWidth + nameWidth + typeWidth + 15 + regionWidth + statusWidth + tagsWidth + 20 // padding
		if totalWidth > opts.MaxWidth {
			// Scale down proportionally
			scale := float64(opts.MaxWidth) / float64(totalWidth)
			idWidth = int(float64(idWidth) * scale)
			nameWidth = int(float64(nameWidth) * scale)
			typeWidth = int(float64(typeWidth) * scale)
			tagsWidth = int(float64(tagsWidth) * scale)
		}
	}
	
	// Create format strings
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%s\n", 
		idWidth, nameWidth, typeWidth, 15, regionWidth, statusWidth)
	rowFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%s\n", 
		idWidth, nameWidth, typeWidth, 15, regionWidth, statusWidth)
	
	// Print header
	if !opts.NoHeader {
		fmt.Printf(headerFormat, "ID", "NAME", "TYPE", "PROVIDER", "REGION", "STATUS", "TAGS")
		
		// Print separator line
		totalHeaderWidth := idWidth + nameWidth + typeWidth + 15 + regionWidth + statusWidth + tagsWidth + 20
		fmt.Println(strings.Repeat("-", totalHeaderWidth))
	}
	
	// Print resources
	for _, resource := range resources {
		// Format tags for display
		var tagStr strings.Builder
		count := 0
		for key, value := range resource.Tags {
			if count > 0 {
				tagStr.WriteString(", ")
			}
			tagStr.WriteString(fmt.Sprintf("%s=%s", key, value))
			count++
			if count >= 3 && !opts.NoTruncate { // Limit to 3 tags unless no-truncate
				if len(resource.Tags) > 3 {
					tagStr.WriteString("...")
				}
				break
			}
		}
		
		// Truncate or keep full values based on options
		var id, name, resourceType, tags string
		if opts.NoTruncate {
			id = resource.ID
			name = resource.Name
			resourceType = resource.Type
			tags = tagStr.String()
		} else {
			id = truncateString(resource.ID, idWidth)
			name = truncateString(resource.Name, nameWidth)
			resourceType = truncateString(resource.Type, typeWidth)
			tags = truncateString(tagStr.String(), tagsWidth)
		}
		
		fmt.Printf(rowFormat, id, name, resourceType, resource.Provider, resource.Region, resource.Status.State, tags)
	}
	
	// Print summary
	fmt.Printf("\nTotal resources: %d\n", len(resources))
	
	// Print helpful tips
	if !opts.NoTruncate && !opts.Wide {
		fmt.Printf("\n💡 Tip: Use --wide or --no-truncate for better readability\n")
		fmt.Printf("   --wide: Wider columns with more spacing\n")
		fmt.Printf("   --no-truncate: Show full names without truncation\n")
	}
	
	return nil
}

// calculateOptimalWidths calculates optimal column widths based on actual data
func calculateOptimalWidths(resources []models.Resource) (int, int, int, int, int, int) {
	maxID, maxName, maxType, maxRegion, maxStatus, maxTags := 10, 10, 10, 10, 10, 10
	
	for _, resource := range resources {
		if len(resource.ID) > maxID {
			maxID = len(resource.ID)
		}
		if len(resource.Name) > maxName {
			maxName = len(resource.Name)
		}
		if len(resource.Type) > maxType {
			maxType = len(resource.Type)
		}
		if len(resource.Region) > maxRegion {
			maxRegion = len(resource.Region)
		}
		if len(resource.Status.State) > maxStatus {
			maxStatus = len(resource.Status.State)
		}
		
		// Calculate tag string length
		var tagStr strings.Builder
		count := 0
		for key, value := range resource.Tags {
			if count > 0 {
				tagStr.WriteString(", ")
			}
			tagStr.WriteString(fmt.Sprintf("%s=%s", key, value))
			count++
		}
		if tagStr.Len() > maxTags {
			maxTags = tagStr.Len()
		}
	}
	
	// Add some padding and reasonable limits
	maxID = min(maxID+2, 50)
	maxName = min(maxName+2, 40)
	maxType = min(maxType+2, 30)
	maxRegion = min(maxRegion+2, 20)
	maxStatus = min(maxStatus+2, 15)
	maxTags = min(maxTags+2, 80)
	
	return maxID, maxName, maxType, maxRegion, maxStatus, maxTags
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// outputInventoryJSON outputs resources in JSON format
func outputInventoryJSON(resources []models.Resource, opts *InventoryOptions) error {
	output := map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	encoder := NewJSONEncoder(os.Stdout)
	return encoder.Encode(output)
}

// outputInventoryYAML outputs resources in YAML format
func outputInventoryYAML(resources []models.Resource, opts *InventoryOptions) error {
	output := map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	encoder := NewYAMLEncoder(os.Stdout)
	return encoder.Encode(output)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}