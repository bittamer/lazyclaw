package gateway

import "github.com/lazyclaw/lazyclaw/internal/models"

// ConnectedMsg is sent when connection is established
type ConnectedMsg struct {
	Scopes          []string
	ProtocolVersion string
	GatewayVersion  string
}

// DisconnectedMsg is sent when connection is lost
type DisconnectedMsg struct {
	Error string
}

// LogMsg is sent when a log event is received
type LogMsg struct {
	Event models.LogEvent
}

// HealthMsg is sent when health data is received
type HealthMsg struct {
	Snapshot models.HealthSnapshot
}
