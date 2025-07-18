package models

import (
	"time"
)

// Resource represents a cloud resource across any provider
type Resource struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Provider  string                 `json:"provider"`
	Region    string                 `json:"region"`
	Status    ResourceStatus         `json:"status"`
	Tags      map[string]string      `json:"tags"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata"`
	Cost      *ResourceCost          `json:"cost,omitempty"`
}

// ResourceType defines common resource types across cloud providers
type ResourceType string

const (
	// Compute resources
	ResourceTypeVirtualMachine ResourceType = "virtual_machine"
	ResourceTypeContainer      ResourceType = "container"
	ResourceTypeFunction       ResourceType = "function"
	ResourceTypeCluster        ResourceType = "cluster"

	// Storage resources
	ResourceTypeObjectStorage ResourceType = "object_storage"
	ResourceTypeBlockStorage  ResourceType = "block_storage"
	ResourceTypeFileStorage   ResourceType = "file_storage"
	ResourceTypeDatabase      ResourceType = "database"

	// Network resources
	ResourceTypeVPC           ResourceType = "vpc"
	ResourceTypeSubnet        ResourceType = "subnet"
	ResourceTypeLoadBalancer  ResourceType = "load_balancer"
	ResourceTypeSecurityGroup ResourceType = "security_group"
	ResourceTypeGateway       ResourceType = "gateway"

	// IAM resources
	ResourceTypeUser   ResourceType = "user"
	ResourceTypeRole   ResourceType = "role"
	ResourceTypePolicy ResourceType = "policy"
	ResourceTypeSecret ResourceType = "secret"

	// Monitoring resources
	ResourceTypeMetric    ResourceType = "metric"
	ResourceTypeAlarm     ResourceType = "alarm"
	ResourceTypeDashboard ResourceType = "dashboard"

	// Other resources
	ResourceTypeUnknown ResourceType = "unknown"
)

// ResourceState defines common resource states
type ResourceState string

const (
	StateRunning    ResourceState = "running"
	StateStopped    ResourceState = "stopped"
	StatePending    ResourceState = "pending"
	StateTerminated ResourceState = "terminated"
	StateError      ResourceState = "error"
	StateUnknown    ResourceState = "unknown"
)

// ResourceHealth defines resource health status
type ResourceHealth string

const (
	HealthHealthy   ResourceHealth = "healthy"
	HealthUnhealthy ResourceHealth = "unhealthy"
	HealthWarning   ResourceHealth = "warning"
	HealthUnknown   ResourceHealth = "unknown"
)

// GetResourceTypeFromString converts a string to ResourceType
func GetResourceTypeFromString(s string) ResourceType {
	switch s {
	case "virtual_machine", "vm", "ec2", "compute_engine", "virtual_machines":
		return ResourceTypeVirtualMachine
	case "container", "containers", "ecs", "gke", "aci":
		return ResourceTypeContainer
	case "function", "functions", "lambda", "cloud_functions", "azure_functions":
		return ResourceTypeFunction
	case "cluster", "clusters", "eks", "gke_cluster", "aks":
		return ResourceTypeCluster
	case "object_storage", "s3", "gcs", "blob_storage":
		return ResourceTypeObjectStorage
	case "block_storage", "ebs", "persistent_disk", "managed_disk":
		return ResourceTypeBlockStorage
	case "file_storage", "efs", "filestore", "azure_files":
		return ResourceTypeFileStorage
	case "database", "databases", "rds", "cloud_sql", "cosmos_db":
		return ResourceTypeDatabase
	case "vpc", "network", "vnet":
		return ResourceTypeVPC
	case "subnet", "subnets":
		return ResourceTypeSubnet
	case "load_balancer", "lb", "elb", "alb", "nlb":
		return ResourceTypeLoadBalancer
	case "security_group", "firewall", "nsg":
		return ResourceTypeSecurityGroup
	case "gateway", "nat_gateway", "internet_gateway":
		return ResourceTypeGateway
	case "user", "users":
		return ResourceTypeUser
	case "role", "roles":
		return ResourceTypeRole
	case "policy", "policies":
		return ResourceTypePolicy
	case "secret", "secrets":
		return ResourceTypeSecret
	case "metric", "metrics":
		return ResourceTypeMetric
	case "alarm", "alarms", "alert":
		return ResourceTypeAlarm
	case "dashboard", "dashboards":
		return ResourceTypeDashboard
	default:
		return ResourceTypeUnknown
	}
}

// String returns the string representation of ResourceType
func (rt ResourceType) String() string {
	return string(rt)
}

// GetStateFromString converts a string to ResourceState
func GetStateFromString(s string) ResourceState {
	switch s {
	case "running", "active", "available", "online":
		return StateRunning
	case "stopped", "inactive", "offline":
		return StateStopped
	case "pending", "starting", "creating", "provisioning":
		return StatePending
	case "terminated", "deleted", "terminating":
		return StateTerminated
	case "error", "failed", "unhealthy":
		return StateError
	default:
		return StateUnknown
	}
}

// GetHealthFromString converts a string to ResourceHealth
func GetHealthFromString(s string) ResourceHealth {
	switch s {
	case "healthy", "ok", "good", "passing":
		return HealthHealthy
	case "unhealthy", "bad", "failing", "critical":
		return HealthUnhealthy
	case "warning", "degraded":
		return HealthWarning
	default:
		return HealthUnknown
	}
}

// NewResource creates a new Resource with default values
func NewResource(id, name, resourceType, provider, region string) *Resource {
	now := time.Now()
	return &Resource{
		ID:       id,
		Name:     name,
		Type:     resourceType,
		Provider: provider,
		Region:   region,
		Status: ResourceStatus{
			State:       string(StateUnknown),
			Health:      string(HealthUnknown),
			LastChecked: now,
		},
		Tags:      make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}
}

// SetTag sets a tag on the resource
func (r *Resource) SetTag(key, value string) {
	if r.Tags == nil {
		r.Tags = make(map[string]string)
	}
	r.Tags[key] = value
}

// GetTag gets a tag value from the resource
func (r *Resource) GetTag(key string) (string, bool) {
	if r.Tags == nil {
		return "", false
	}
	value, exists := r.Tags[key]
	return value, exists
}

// SetMetadata sets metadata on the resource
func (r *Resource) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
}

// GetMetadata gets metadata from the resource
func (r *Resource) GetMetadata(key string) (interface{}, bool) {
	if r.Metadata == nil {
		return nil, false
	}
	value, exists := r.Metadata[key]
	return value, exists
}

// UpdateStatus updates the resource status
func (r *Resource) UpdateStatus(state string, health string) {
	r.Status.State = state
	r.Status.Health = health
	r.Status.LastChecked = time.Now()
	r.UpdatedAt = time.Now()
}
