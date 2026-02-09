package gateway

import (
	"math/rand"
	"time"

	"github.com/lazyclaw/lazyclaw/internal/models"
)

// MockClient simulates a gateway connection for UI testing
type MockClient struct {
	connected bool
	logs      chan models.LogEvent
	done      chan struct{}
}

// NewMockClient creates a mock gateway client
func NewMockClient() *MockClient {
	return &MockClient{
		logs: make(chan models.LogEvent, 100),
		done: make(chan struct{}),
	}
}

// Connect simulates connecting to a gateway
func (m *MockClient) Connect() interface{} {
	m.connected = true

	// Start generating mock logs
	go m.generateMockLogs()

	return ConnectedMsg{
		Scopes:          []string{"operator.read"},
		ProtocolVersion: "1",
		GatewayVersion:  "mock-1.0.0",
	}
}

// Close closes the mock client
func (m *MockClient) Close() error {
	close(m.done)
	m.connected = false
	return nil
}

// GetLogs returns the log channel
func (m *MockClient) GetLogs() <-chan models.LogEvent {
	return m.logs
}

func (m *MockClient) generateMockLogs() {
	messages := []struct {
		level   string
		message string
	}{
		{"info", "Gateway started successfully"},
		{"info", "WhatsApp channel connected"},
		{"info", "Telegram channel connected"},
		{"debug", "Heartbeat sent"},
		{"info", "New session started: user_123"},
		{"debug", "Processing incoming message"},
		{"info", "Agent 'assistant' handling request"},
		{"debug", "Tool call: web_search"},
		{"info", "Response sent to user"},
		{"warn", "Rate limit approaching for API calls"},
		{"info", "Session compaction triggered"},
		{"debug", "Cache hit for embedding lookup"},
		{"info", "Webhook received from external service"},
		{"error", "Failed to connect to backup server (retrying...)"},
		{"info", "Backup server connection restored"},
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			msg := messages[rand.Intn(len(messages))]
			select {
			case m.logs <- models.LogEvent{
				Timestamp: time.Now(),
				Level:     msg.level,
				Source:    "gateway",
				Message:   msg.message,
			}:
			default:
				// Channel full, skip
			}
		}
	}
}

// GetMockHealth returns mock health data
func GetMockHealth() *models.HealthSnapshot {
	return &models.HealthSnapshot{
		Timestamp:       time.Now(),
		ProbeDurationMs: 45,
		ChannelSummary: map[string]models.ChannelHealth{
			"whatsapp": {
				Name:      "whatsapp",
				Connected: true,
				AuthAge:   24 * time.Hour,
			},
			"telegram": {
				Name:      "telegram",
				Connected: true,
				AuthAge:   48 * time.Hour,
			},
		},
		SessionCount: 12,
	}
}
