package apitypes

import (
	"pressluft/internal/controlplane/activity"
)

type ActivityListResponse struct {
	Data       []Activity `json:"data"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type Activity struct {
	ID                 string                `json:"id"`
	EventType          activity.EventType    `json:"event_type"`
	Category           activity.Category     `json:"category"`
	Level              activity.Level        `json:"level"`
	ResourceType       activity.ResourceType `json:"resource_type,omitempty"`
	ResourceID         string                `json:"resource_id,omitempty"`
	ParentResourceType activity.ResourceType `json:"parent_resource_type,omitempty"`
	ParentResourceID   string                `json:"parent_resource_id,omitempty"`
	ActorType          activity.ActorType    `json:"actor_type"`
	ActorID            string                `json:"actor_id,omitempty"`
	Title              string                `json:"title"`
	Message            string                `json:"message,omitempty"`
	Payload            string                `json:"payload,omitempty"`
	RequiresAttention  bool                  `json:"requires_attention"`
	ReadAt             string                `json:"read_at,omitempty"`
	CreatedAt          string                `json:"created_at"`
}

func APIActivity(in activity.Activity) Activity {
	return Activity{
		ID:                 in.ID,
		EventType:          in.EventType,
		Category:           in.Category,
		Level:              in.Level,
		ResourceType:       in.ResourceType,
		ResourceID:         FormatAppID(in.ResourceID),
		ParentResourceType: in.ParentResourceType,
		ParentResourceID:   FormatAppID(in.ParentResourceID),
		ActorType:          in.ActorType,
		ActorID:            in.ActorID,
		Title:              in.Title,
		Message:            in.Message,
		Payload:            in.Payload,
		RequiresAttention:  in.RequiresAttention,
		ReadAt:             in.ReadAt,
		CreatedAt:          in.CreatedAt,
	}
}

func APIActivities(in []activity.Activity) []Activity {
	out := make([]Activity, 0, len(in))
	for _, item := range in {
		out = append(out, APIActivity(item))
	}
	return out
}
