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

// TableColumnWidths holds the calculated column widths
type TableColumnWidths struct {
	ID       int
	Name     int
	Type     int
	Provider int
	Region   int
	Status   int
	Tags     int
}

// calculateColumnWidths calculates optimal column widths based on data and options
func calculateColumnWidths(resources []models.Resource, opts *InventoryOptions) TableColumnWidths {
	// Start with minimum header widths
	widths := TableColumnWidths{
		ID:       2,  // "ID"
		Name:     4,  // "NAME"
		Type:     4,  // "TYPE"
		Provider: 8,  // "PROVIDER"
		Region:   6,  // "REGION"
		Status:   6,  // "STATUS"
		Tags:     4,  // "TAGS"
	}

	// Calculate based on actual data
	for _, resource := range resources {
		if len(resource.ID) > widths.ID {
			widths.ID = len(resource.ID)
		}
		if len(resource.Name) > widths.Name {
			widths.Name = len(resource.Name)
		}
		if len(resource.Type) > widths.Type {
			widths.Type = len(resource.Type)
		}
		if len(resource.Provider) > widths.Provider {
			widths.Provider = len(resource.Provider)
		}
		if len(resource.Region) > widths.Region {
			widths.Region = len(resource.Region)
		}
		if len(resource.Status.State) > widths.Status {
			widths.Status = len(resource.Status.State)
		}

		// Calculate tag string length
		tagStr := formatTagsForDisplay(resource.Tags, opts.NoTruncate)
		if len(tagStr) > widths.Tags {
			widths.Tags = len(tagStr)
		}
	}

	// Apply formatting rules based on options
	if opts.NoTruncate {
		// For no-truncate, use calculated widths with some padding
		widths.ID += 2
		widths.Name += 2
		widths.Type += 2
		widths.Provider += 2
		widths.Region += 2
		widths.Status += 2
		widths.Tags += 2
	} else if opts.Wide {
		// Wide format: generous but reasonable limits
		widths.ID = maxInt(widths.ID, 45)
		widths.Name = maxInt(widths.Name, 35)
		widths.Type = maxInt(widths.Type, 20)
		widths.Provider = maxInt(widths.Provider, 12)
		widths.Region = maxInt(widths.Region, 12)
		widths.Status = maxInt(widths.Status, 12)
		widths.Tags = maxInt(widths.Tags, 50)

		// Apply reasonable maximums for wide mode
		widths.ID = minInt(widths.ID, 60)
		widths.Name = minInt(widths.Name, 50)
		widths.Tags = minInt(widths.Tags, 80)
	} else {
		// Standard format: compact but readable
		widths.ID = maxInt(minInt(widths.ID, 25), 12)
		widths.Name = maxInt(minInt(widths.Name, 25), 12)
		widths.Type = maxInt(minInt(widths.Type, 18), 12)
		widths.Provider = maxInt(minInt(widths.Provider, 12), 8)
		widths.Region = maxInt(minInt(widths.Region, 12), 8)
		widths.Status = maxInt(minInt(widths.Status, 12), 8)
		widths.Tags = maxInt(minInt(widths.Tags, 40), 8)
	}

	// Apply max width constraint if specified
	if opts.MaxWidth > 0 {
		totalWidth := widths.ID + widths.Name + widths.Type + widths.Provider + widths.Region + widths.Status + widths.Tags + 18 // 6 spaces between 7 columns
		if totalWidth > opts.MaxWidth {
			// Scale down proportionally, but preserve minimums
			scale := float64(opts.MaxWidth-50) / float64(totalWidth-50) // Reserve 50 chars for minimums
			if scale > 0 && scale < 1 {
				widths.ID = maxInt(int(float64(widths.ID)*scale), 8)
				widths.Name = maxInt(int(float64(widths.Name)*scale), 8)
				widths.Type = maxInt(int(float64(widths.Type)*scale), 8)
				widths.Tags = maxInt(int(float64(widths.Tags)*scale), 8)
			}
		}
	}

	return widths
}

// formatTagsForDisplay formats tags for table display
func formatTagsForDisplay(tags map[string]string, noTruncate bool) string {
	if len(tags) == 0 {
		return ""
	}

	var tagPairs []string
	for key, value := range tags {
		tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", key, value))
	}

	result := strings.Join(tagPairs, ", ")

	// Apply truncation if needed
	if !noTruncate && len(result) > 40 {
		if len(tagPairs) > 1 {
			result = tagPairs[0] + ", ..."
		} else {
			result = truncateString(result, 40)
		}
	}

	return result
}

// outputInventoryTable outputs resources in an improved table format
func outputInventoryTable(resources []models.Resource, opts *InventoryOptions) error {
	if len(resources) == 0 {
		fmt.Println("No resources found.")
		return nil
	}

	// Calculate column widths
	widths := calculateColumnWidths(resources, opts)

	// Create format strings for proper alignment
	headerFormat := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n",
		widths.ID, widths.Name, widths.Type, widths.Provider, widths.Region, widths.Status)

	rowFormat := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n",
		widths.ID, widths.Name, widths.Type, widths.Provider, widths.Region, widths.Status)

	// Print header
	if !opts.NoHeader {
		fmt.Printf(headerFormat, "ID", "NAME", "TYPE", "PROVIDER", "REGION", "STATUS", "TAGS")

		// Print separator line
		separator := strings.Repeat("-", widths.ID) + "  " +
			strings.Repeat("-", widths.Name) + "  " +
			strings.Repeat("-", widths.Type) + "  " +
			strings.Repeat("-", widths.Provider) + "  " +
			strings.Repeat("-", widths.Region) + "  " +
			strings.Repeat("-", widths.Status) + "  " +
			strings.Repeat("-", widths.Tags)
		fmt.Println(separator)
	}

	// Print resources
	for _, resource := range resources {
		// Prepare display values
		id := prepareDisplayValue(resource.ID, widths.ID, opts.NoTruncate)
		name := prepareDisplayValue(resource.Name, widths.Name, opts.NoTruncate)
		resourceType := prepareDisplayValue(resource.Type, widths.Type, opts.NoTruncate)
		provider := prepareDisplayValue(resource.Provider, widths.Provider, opts.NoTruncate)
		region := prepareDisplayValue(resource.Region, widths.Region, opts.NoTruncate)
		status := prepareDisplayValue(resource.Status.State, widths.Status, opts.NoTruncate)
		tags := formatTagsForDisplay(resource.Tags, opts.NoTruncate)

		fmt.Printf(rowFormat, id, name, resourceType, provider, region, status, tags)
	}

	// Print summary
	fmt.Printf("\nTotal resources: %d\n", len(resources))

	// Print helpful tips if using default formatting
	if !opts.NoTruncate && !opts.Wide {
		fmt.Printf("\n💡 Tip: Use --wide or --no-truncate for better readability\n")
		fmt.Printf("   --wide: Wider columns with more spacing\n")
		fmt.Printf("   --no-truncate: Show full names without truncation\n")
	}

	return nil
}

// prepareDisplayValue prepares a value for display, applying truncation if needed
func prepareDisplayValue(value string, maxWidth int, noTruncate bool) string {
	if noTruncate || len(value) <= maxWidth {
		return value
	}
	return truncateString(value, maxWidth)
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

// Helper functions for min/max
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
