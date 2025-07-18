package types

import "time"

// ResourceFilters defines filters for resource queries
type ResourceFilters struct {
	Regions       []string          `json:"regions,omitempty"`
	ResourceTypes []string          `json:"resource_types,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Status        []string          `json:"status,omitempty"`
	CreatedAfter  *time.Time        `json:"created_after,omitempty"`
	CreatedBefore *time.Time        `json:"created_before,omitempty"`
}

// CostPeriod defines the time period for cost queries
type CostPeriod struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Granularity string    `json:"granularity"` // DAILY, WEEKLY, MONTHLY
}

// AlertFilters defines filters for alert queries
type AlertFilters struct {
	Severity     []string   `json:"severity,omitempty"`
	Status       []string   `json:"status,omitempty"`
	ResourceID   string     `json:"resource_id,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
}

// SecurityFilters defines filters for security finding queries
type SecurityFilters struct {
	Severity   []string `json:"severity,omitempty"`
	Category   []string `json:"category,omitempty"`
	ResourceID string   `json:"resource_id,omitempty"`
	Framework  string   `json:"framework,omitempty"`
}