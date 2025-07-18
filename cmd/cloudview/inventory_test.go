package cloudview

import (
	"testing"
	"time"

	"github.com/Tsahi-Elkayam/cloudview/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInventoryFilters(t *testing.T) {
	tests := []struct {
		name    string
		opts    *InventoryOptions
		want    types.ResourceFilters
		wantErr bool
	}{
		{
			name: "basic filters",
			opts: &InventoryOptions{
				Providers:     []string{"aws"},
				Regions:       []string{"us-east-1", "us-west-2"},
				ResourceTypes: []string{"ec2", "s3"},
				Status:        []string{"running", "available"},
			},
			want: types.ResourceFilters{
				Regions:       []string{"us-east-1", "us-west-2"},
				ResourceTypes: []string{"ec2", "s3"},
				Status:        []string{"running", "available"},
				Tags:          make(map[string]string),
			},
			wantErr: false,
		},
		{
			name: "with tags",
			opts: &InventoryOptions{
				Tags: []string{"Environment=production", "Team=backend"},
			},
			want: types.ResourceFilters{
				Tags: map[string]string{
					"Environment": "production",
					"Team":        "backend",
				},
			},
			wantErr: false,
		},
		{
			name: "with date filters",
			opts: &InventoryOptions{
				CreatedAfter:  "2024-01-01",
				CreatedBefore: "2024-01-31",
			},
			want: func() types.ResourceFilters {
				after, _ := time.Parse("2006-01-02", "2024-01-01")
				before, _ := time.Parse("2006-01-02", "2024-01-31")
				return types.ResourceFilters{
					CreatedAfter:  &after,
					CreatedBefore: &before,
					Tags:          make(map[string]string),
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid tag format",
			opts: &InventoryOptions{
				Tags: []string{"invalid-tag-format"},
			},
			wantErr: true,
		},
		{
			name: "invalid date format",
			opts: &InventoryOptions{
				CreatedAfter: "invalid-date",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInventoryFilters(tt.opts)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.want.Regions, got.Regions)
			assert.Equal(t, tt.want.ResourceTypes, got.ResourceTypes)
			assert.Equal(t, tt.want.Status, got.Status)
			assert.Equal(t, tt.want.Tags, got.Tags)
			
			if tt.want.CreatedAfter != nil {
				require.NotNil(t, got.CreatedAfter)
				assert.Equal(t, *tt.want.CreatedAfter, *got.CreatedAfter)
			}
			
			if tt.want.CreatedBefore != nil {
				require.NotNil(t, got.CreatedBefore)
				assert.Equal(t, *tt.want.CreatedBefore, *got.CreatedBefore)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string",
			input:  "this is a very long string that needs to be truncated",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "very short max length",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Integration-style test to verify the inventory command structure
func TestInventoryCommandCreation(t *testing.T) {
	// This test verifies the command can be created without errors
	logger := &MockLogger{}
	cmd := NewInventoryCommand(logger)
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "inventory", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	
	// Check that flags are properly defined
	flags := cmd.Flags()
	
	// Check provider flag
	providerFlag := flags.Lookup("provider")
	assert.NotNil(t, providerFlag)
	assert.Equal(t, "p", providerFlag.Shorthand)
	
	// Check region flag
	regionFlag := flags.Lookup("region")
	assert.NotNil(t, regionFlag)
	assert.Equal(t, "r", regionFlag.Shorthand)
	
	// Check type flag
	typeFlag := flags.Lookup("type")
	assert.NotNil(t, typeFlag)
	assert.Equal(t, "t", typeFlag.Shorthand)
	
	// Check output flag
	outputFlag := flags.Lookup("output")
	assert.NotNil(t, outputFlag)
	assert.Equal(t, "o", outputFlag.Shorthand)
}

// MockLogger implements the basic logging interface for testing
type MockLogger struct {
	lastLevel string
	messages  []string
}

func (m *MockLogger) SetLevel(level interface{}) {
	// For testing, we just track that SetLevel was called
	m.lastLevel = "debug"
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+format)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.messages = append(m.messages, "INFO: "+format)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.messages = append(m.messages, "WARN: "+format)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+format)
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.messages = append(m.messages, "FATAL")
}

// GetMessages returns all logged messages for testing
func (m *MockLogger) GetMessages() []string {
	return m.messages
}

// ClearMessages clears all logged messages
func (m *MockLogger) ClearMessages() {
	m.messages = []string{}
}