package models

import "time"

// ResourceStatus represents the current status of a resource
type ResourceStatus struct {
	State       string    `json:"state"`
	Health      string    `json:"health"`
	LastChecked time.Time `json:"last_checked"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	Daily    float64 `json:"daily"`
	Monthly  float64 `json:"monthly"`
	Currency string  `json:"currency"`
}

// Cost represents cost data for cloud resources
type Cost struct {
	Provider   string            `json:"provider"`
	Service    string            `json:"service"`
	ResourceID string            `json:"resource_id,omitempty"`
	Amount     float64           `json:"amount"`
	Currency   string            `json:"currency"`
	Period     string            `json:"period"`
	Date       time.Time         `json:"date"`
	Dimensions map[string]string `json:"dimensions"`
}

// ServiceCost represents aggregated cost by service
type ServiceCost struct {
	Provider string  `json:"provider"`
	Service  string  `json:"service"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Period   string  `json:"period"`
}

// CostForecast represents predicted future costs
type CostForecast struct {
	Provider string    `json:"provider"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
}

// Metric represents monitoring metrics
type Metric struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Labels    map[string]string      `json:"labels"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus defines alert status values
type AlertStatus string

const (
	StatusOpen         AlertStatus = "open"
	StatusAcknowledged AlertStatus = "acknowledged"
	StatusResolved     AlertStatus = "resolved"
	StatusSuppressed   AlertStatus = "suppressed"
)

// Alert represents a monitoring alert
type Alert struct {
	ID          string            `json:"id"`
	Provider    string            `json:"provider"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    AlertSeverity     `json:"severity"`
	Status      AlertStatus       `json:"status"`
	ResourceID  string            `json:"resource_id,omitempty"`
	Tags        map[string]string `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SecurityFinding represents a security finding or vulnerability
type SecurityFinding struct {
	ID          string           `json:"id"`
	Provider    string           `json:"provider"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Severity    AlertSeverity    `json:"severity"`
	Category    string           `json:"category"`
	ResourceID  string           `json:"resource_id"`
	Region      string           `json:"region"`
	Compliance  []ComplianceInfo `json:"compliance"`
	CreatedAt   time.Time        `json:"created_at"`
}

// ComplianceInfo represents compliance framework information
type ComplianceInfo struct {
	Framework string `json:"framework"`
	Control   string `json:"control"`
	Status    string `json:"status"`
}

// ComplianceResult represents the result of a compliance check
type ComplianceResult struct {
	Framework   string    `json:"framework"`
	Control     string    `json:"control"`
	Status      string    `json:"status"`
	Score       float64   `json:"score"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// Recommendation represents optimization or security recommendations
type Recommendation struct {
	ID          string            `json:"id"`
	Provider    string            `json:"provider"`
	Category    string            `json:"category"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Impact      string            `json:"impact"`
	Effort      string            `json:"effort"`
	Savings     *float64          `json:"savings,omitempty"`
	ResourceID  string            `json:"resource_id,omitempty"`
	Actions     []string          `json:"actions"`
	Tags        map[string]string `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page    int  `json:"page"`
	PerPage int  `json:"per_page"`
	Total   int  `json:"total"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

// Result represents a generic result with data and metadata
type Result struct {
	Data       interface{}            `json:"data"`
	Pagination *PaginationInfo        `json:"pagination,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
