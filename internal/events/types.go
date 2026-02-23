package events

// StreamEnvelope is the wire shape for server-sent event payloads.
type StreamEnvelope struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}
