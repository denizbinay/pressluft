package activity

import "fmt"

// Level represents the severity of an activity entry.
type Level string

const (
	LevelInfo    Level = "info"
	LevelSuccess Level = "success"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

// Category groups related activity types.
type Category string

const (
	CategoryJob      Category = "job"
	CategoryServer   Category = "server"
	CategoryProvider Category = "provider"
	CategorySite     Category = "site"
	CategoryAccount  Category = "account"
	CategorySecurity Category = "security"
)

// ActorType identifies who triggered the activity.
type ActorType string

const (
	ActorSystem ActorType = "system"
	ActorUser   ActorType = "user"
	ActorAPI    ActorType = "api"
)

// EventType is the specific event identifier.
// Only these event types are allowed.
type EventType string

// Job events
const (
	EventJobCreated   EventType = "job.created"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"
	EventJobCancelled EventType = "job.cancelled"
)

// Server events
const (
	EventServerCreated       EventType = "server.created"
	EventServerProvisioned   EventType = "server.provisioned"
	EventServerDeleted       EventType = "server.deleted"
	EventServerStatusChanged EventType = "server.status_changed"
)

// Provider events
const (
	EventProviderAdded   EventType = "provider.added"
	EventProviderUpdated EventType = "provider.updated"
	EventProviderRemoved EventType = "provider.removed"
)

// Site events
const (
	EventSiteCreated  EventType = "site.created"
	EventSiteDeployed EventType = "site.deployed"
	EventSiteDeleted  EventType = "site.deleted"
)

// Account events
const (
	EventAccountSettingsChanged EventType = "account.settings_changed"
)

// Security events
const (
	EventSecurityAPIKeyCreated EventType = "security.api_key_created"
	EventSecurityAPIKeyRevoked EventType = "security.api_key_revoked"
)

// validEventTypes is the set of all allowed event types.
var validEventTypes = map[EventType]bool{
	// Job events
	EventJobCreated:   true,
	EventJobStarted:   true,
	EventJobCompleted: true,
	EventJobFailed:    true,
	EventJobCancelled: true,
	// Server events
	EventServerCreated:       true,
	EventServerProvisioned:   true,
	EventServerDeleted:       true,
	EventServerStatusChanged: true,
	// Provider events
	EventProviderAdded:   true,
	EventProviderUpdated: true,
	EventProviderRemoved: true,
	// Site events
	EventSiteCreated:  true,
	EventSiteDeployed: true,
	EventSiteDeleted:  true,
	// Account events
	EventAccountSettingsChanged: true,
	// Security events
	EventSecurityAPIKeyCreated: true,
	EventSecurityAPIKeyRevoked: true,
}

// ValidateEventType checks if the given event type is valid.
func ValidateEventType(et EventType) error {
	if !validEventTypes[et] {
		return fmt.Errorf("unknown event type: %s", et)
	}
	return nil
}

// ResourceType identifies the type of resource.
type ResourceType string

const (
	ResourceJob      ResourceType = "job"
	ResourceServer   ResourceType = "server"
	ResourceProvider ResourceType = "provider"
	ResourceSite     ResourceType = "site"
	ResourceAccount  ResourceType = "account"
	ResourceAPIKey   ResourceType = "api_key"
)

// Activity is a single entry in the activity stream.
type Activity struct {
	ID int64 `json:"id"`

	// Event classification
	EventType EventType `json:"event_type"`
	Category  Category  `json:"category"`
	Level     Level     `json:"level"`

	// Polymorphic resource reference
	ResourceType ResourceType `json:"resource_type,omitempty"`
	ResourceID   int64        `json:"resource_id,omitempty"`

	// Secondary resource (e.g., job belongs to server)
	ParentResourceType ResourceType `json:"parent_resource_type,omitempty"`
	ParentResourceID   int64        `json:"parent_resource_id,omitempty"`

	// Actor tracking
	ActorType ActorType `json:"actor_type"`
	ActorID   string    `json:"actor_id,omitempty"`

	// Content
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
	Payload string `json:"payload,omitempty"`

	// Notification projection
	RequiresAttention bool   `json:"requires_attention"`
	ReadAt            string `json:"read_at,omitempty"`

	CreatedAt string `json:"created_at"`
}

// EmitInput is the input for creating a new activity entry.
type EmitInput struct {
	EventType EventType
	Category  Category
	Level     Level

	ResourceType ResourceType
	ResourceID   int64

	ParentResourceType ResourceType
	ParentResourceID   int64

	ActorType ActorType
	ActorID   string

	Title   string
	Message string
	Payload string

	RequiresAttention bool
}

// ListFilter contains filtering and pagination options for listing activity.
type ListFilter struct {
	// Cursor-based pagination (use ID)
	Cursor int64
	Limit  int

	// Filtering
	Category           Category
	ResourceType       ResourceType
	ResourceID         int64
	ParentResourceType ResourceType
	ParentResourceID   int64
	RequiresAttention  *bool
	UnreadOnly         bool
}
