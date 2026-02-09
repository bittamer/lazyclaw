package models

import "time"

// ConnectionMode defines how to connect to an OpenClaw Gateway
type ConnectionMode string

const (
	ConnectionModeLocal ConnectionMode = "local" // Run openclaw locally
	ConnectionModeSSH   ConnectionMode = "ssh"   // Run openclaw via SSH on remote host
)

// HealthLevel indicates the overall health status of an instance
type HealthLevel string

const (
	HealthOK       HealthLevel = "OK"
	HealthDegraded HealthLevel = "DEGRADED"
	HealthDown     HealthLevel = "DOWN"
)

// InstanceProfile represents a configured OpenClaw Gateway instance
type InstanceProfile struct {
	Name        string         `yaml:"name" json:"name"`
	Tags        []string       `yaml:"tags,omitempty" json:"tags,omitempty"`
	Mode        ConnectionMode `yaml:"mode" json:"mode"`
	SSH         *SSHConfig     `yaml:"ssh,omitempty" json:"ssh,omitempty"`
	OpenClawCLI string         `yaml:"openclaw_cli,omitempty" json:"openclaw_cli,omitempty"` // Path to openclaw on remote/local
}

// SSHConfig holds SSH connection configuration for remote instances
type SSHConfig struct {
	Host           string `yaml:"host" json:"host"`                                           // SSH host (e.g., "user@hostname" or "hostname")
	Port           int    `yaml:"port,omitempty" json:"port,omitempty"`                       // SSH port (default: 22)
	User           string `yaml:"user,omitempty" json:"user,omitempty"`                       // SSH user (optional if in host)
	IdentityFile   string `yaml:"identity_file,omitempty" json:"identity_file,omitempty"`     // Path to SSH private key
	ProxyJump      string `yaml:"proxy_jump,omitempty" json:"proxy_jump,omitempty"`           // SSH proxy/jump host
	ConnectTimeout int    `yaml:"connect_timeout,omitempty" json:"connect_timeout,omitempty"` // Connection timeout in seconds
	OpenClawCLI    string `yaml:"openclaw_cli,omitempty" json:"openclaw_cli,omitempty"`       // Path to openclaw binary on remote host
}

// ConnectionState tracks the current connection status
type ConnectionState struct {
	Connected       bool
	LastError       string
	LastHandshake   time.Time
	Scopes          []string
	ProtocolVersion string
	GatewayVersion  string
}

// LogEvent represents a single log entry from the gateway
type LogEvent struct {
	Timestamp time.Time
	Level     string // debug, info, warn, error
	Source    string
	Message   string
	Raw       string
}

// HealthSnapshot contains gateway health information
type HealthSnapshot struct {
	Timestamp       time.Time
	ProbeDurationMs int64
	ChannelSummary  map[string]ChannelHealth
	SessionCount    int
	LastError       string
}

// ChannelHealth represents the health of a single channel
type ChannelHealth struct {
	Name      string
	Connected bool
	LastError string
	AuthAge   time.Duration
}

// GatewayStatus holds overall gateway status
type GatewayStatus struct {
	Reachable    bool
	Mode         string
	Uptime       time.Duration
	Version      string
	SessionCount int
	ChannelCount int
	ActiveAgents int
}

// ============================================================================
// OpenClaw Health JSON structures (from `openclaw health --json`)
// ============================================================================

// HealthCheckResult represents the full output of `openclaw health --json`
type HealthCheckResult struct {
	Overall        string              `json:"overall"`        // "ok", "degraded", "down"
	Timestamp      int64               `json:"ts,omitempty"`
	Gateway        *HealthGateway      `json:"gateway,omitempty"`
	Channels       []HealthChannelItem `json:"channels,omitempty"`
	Services       []HealthServiceItem `json:"services,omitempty"`
	Doctor         []HealthDoctorItem  `json:"doctor,omitempty"`
	ProbeDurationMs int64              `json:"probeDurationMs,omitempty"`
	Raw            string              `json:"-"` // Raw JSON for fallback display
}

// HealthGateway contains gateway health info
type HealthGateway struct {
	Reachable       bool   `json:"reachable"`
	LatencyMs       int    `json:"latencyMs,omitempty"`
	Version         string `json:"version,omitempty"`
	Error           string `json:"error,omitempty"`
}

// HealthChannelItem contains health info for a single channel
type HealthChannelItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"`    // "ok", "error", "warning", "unknown"
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
	AuthAgeMs int64  `json:"authAgeMs,omitempty"`
}

// HealthServiceItem contains health info for a system service
type HealthServiceItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "running", "stopped", "not_installed"
	Details string `json:"details,omitempty"`
}

// HealthDoctorItem contains a diagnostic finding from `openclaw doctor`
type HealthDoctorItem struct {
	Check   string `json:"check"`
	Status  string `json:"status"` // "pass", "warn", "fail"
	Message string `json:"message"`
}

// ============================================================================
// OpenClaw Status JSON structures (from `openclaw status --json`)
// ============================================================================

// OpenClawStatus represents the full output of `openclaw status --json`
type OpenClawStatus struct {
	LinkChannel    *LinkChannel    `json:"linkChannel,omitempty"`
	Heartbeat      *Heartbeat      `json:"heartbeat,omitempty"`
	ChannelSummary []string        `json:"channelSummary,omitempty"`
	Sessions       *Sessions       `json:"sessions,omitempty"`
	OS             *OSInfo         `json:"os,omitempty"`
	Update         *UpdateInfo     `json:"update,omitempty"`
	UpdateChannel  string          `json:"updateChannel,omitempty"`
	Memory         *MemoryInfo     `json:"memory,omitempty"`
	Gateway        *GatewayInfo    `json:"gateway,omitempty"`
	GatewayService *ServiceInfo    `json:"gatewayService,omitempty"`
	NodeService    *ServiceInfo    `json:"nodeService,omitempty"`
	Agents         *AgentsInfo     `json:"agents,omitempty"`
	SecurityAudit  *SecurityAudit  `json:"securityAudit,omitempty"`
}

// LinkChannel represents the linked channel status (e.g., WhatsApp)
type LinkChannel struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Linked    bool    `json:"linked"`
	AuthAgeMs float64 `json:"authAgeMs"`
}

// Heartbeat contains heartbeat configuration
type Heartbeat struct {
	DefaultAgentID string            `json:"defaultAgentId"`
	Agents         []HeartbeatAgent  `json:"agents"`
}

// HeartbeatAgent represents a heartbeat agent configuration
type HeartbeatAgent struct {
	AgentID string `json:"agentId"`
	Enabled bool   `json:"enabled"`
	Every   string `json:"every"`
	EveryMs int64  `json:"everyMs"`
}

// Sessions contains session information
type Sessions struct {
	Paths    []string       `json:"paths"`
	Count    int            `json:"count"`
	Defaults SessionDefault `json:"defaults"`
	Recent   []Session      `json:"recent"`
	ByAgent  []AgentSession `json:"byAgent"`
}

// SessionDefault contains default session settings
type SessionDefault struct {
	Model         string `json:"model"`
	ContextTokens int    `json:"contextTokens"`
}

// Session represents a single session
type Session struct {
	AgentID         string   `json:"agentId"`
	Key             string   `json:"key"`
	Kind            string   `json:"kind"` // "direct", "group"
	SessionID       string   `json:"sessionId"`
	UpdatedAt       int64    `json:"updatedAt"`
	Age             int64    `json:"age"`
	SystemSent      bool     `json:"systemSent"`
	AbortedLastRun  bool     `json:"abortedLastRun,omitempty"`
	InputTokens     int      `json:"inputTokens"`
	OutputTokens    int      `json:"outputTokens"`
	TotalTokens     int      `json:"totalTokens"`
	RemainingTokens int      `json:"remainingTokens"`
	PercentUsed     int      `json:"percentUsed"`
	Model           string   `json:"model"`
	ContextTokens   int      `json:"contextTokens"`
	Flags           []string `json:"flags"`
}

// AgentSession groups sessions by agent
type AgentSession struct {
	AgentID string    `json:"agentId"`
	Path    string    `json:"path"`
	Count   int       `json:"count"`
	Recent  []Session `json:"recent"`
}

// OSInfo contains operating system information
type OSInfo struct {
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Release  string `json:"release"`
	Label    string `json:"label"`
}

// UpdateInfo contains update status information
type UpdateInfo struct {
	Root           string       `json:"root"`
	InstallKind    string       `json:"installKind"`
	PackageManager string       `json:"packageManager"`
	Deps           DepsInfo     `json:"deps"`
	Registry       RegistryInfo `json:"registry"`
}

// DepsInfo contains dependency status
type DepsInfo struct {
	Manager      string `json:"manager"`
	Status       string `json:"status"`
	LockfilePath string `json:"lockfilePath"`
	MarkerPath   string `json:"markerPath"`
	Reason       string `json:"reason,omitempty"`
}

// RegistryInfo contains registry version info
type RegistryInfo struct {
	LatestVersion string `json:"latestVersion"`
}

// MemoryInfo contains memory/RAG system information
type MemoryInfo struct {
	AgentID           string        `json:"agentId"`
	Backend           string        `json:"backend"`
	Files             int           `json:"files"`
	Chunks            int           `json:"chunks"`
	Dirty             bool          `json:"dirty"`
	WorkspaceDir      string        `json:"workspaceDir"`
	DBPath            string        `json:"dbPath"`
	Provider          string        `json:"provider"`
	Model             string        `json:"model"`
	RequestedProvider string        `json:"requestedProvider"`
	Sources           []string      `json:"sources"`
	SourceCounts      []SourceCount `json:"sourceCounts"`
	Cache             CacheInfo     `json:"cache"`
	FTS               FTSInfo       `json:"fts"`
	Vector            VectorInfo    `json:"vector"`
}

// SourceCount contains source file/chunk counts
type SourceCount struct {
	Source string `json:"source"`
	Files  int    `json:"files"`
	Chunks int    `json:"chunks"`
}

// CacheInfo contains cache status
type CacheInfo struct {
	Enabled bool `json:"enabled"`
	Entries int  `json:"entries"`
}

// FTSInfo contains full-text search status
type FTSInfo struct {
	Enabled   bool `json:"enabled"`
	Available bool `json:"available"`
}

// VectorInfo contains vector search status
type VectorInfo struct {
	Enabled       bool   `json:"enabled"`
	Available     bool   `json:"available"`
	ExtensionPath string `json:"extensionPath,omitempty"`
	Dims          int    `json:"dims,omitempty"`
}

// GatewayInfo contains gateway connection information
type GatewayInfo struct {
	Mode             string      `json:"mode"`
	URL              string      `json:"url"`
	URLSource        string      `json:"urlSource"`
	Misconfigured    bool        `json:"misconfigured"`
	Reachable        bool        `json:"reachable"`
	ConnectLatencyMs int         `json:"connectLatencyMs"`
	Self             GatewaySelf `json:"self"`
	Error            *string     `json:"error"`
}

// GatewaySelf contains gateway self-identification
type GatewaySelf struct {
	Host     string `json:"host"`
	IP       string `json:"ip"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// ServiceInfo contains systemd service status
type ServiceInfo struct {
	Label        string `json:"label"`
	Installed    bool   `json:"installed"`
	LoadedText   string `json:"loadedText"`
	RuntimeShort string `json:"runtimeShort"`
}

// AgentsInfo contains agent information
type AgentsInfo struct {
	DefaultID             string      `json:"defaultId"`
	Agents                []AgentInfo `json:"agents"`
	TotalSessions         int         `json:"totalSessions"`
	BootstrapPendingCount int         `json:"bootstrapPendingCount"`
}

// AgentInfo contains information about a single agent
type AgentInfo struct {
	ID               string `json:"id"`
	WorkspaceDir     string `json:"workspaceDir"`
	BootstrapPending bool   `json:"bootstrapPending"`
	SessionsPath     string `json:"sessionsPath"`
	SessionsCount    int    `json:"sessionsCount"`
	LastUpdatedAt    int64  `json:"lastUpdatedAt"`
	LastActiveAgeMs  int64  `json:"lastActiveAgeMs"`
}

// SecurityAudit contains security audit results
type SecurityAudit struct {
	Timestamp int64                  `json:"ts"`
	Summary   SecurityAuditSummary   `json:"summary"`
	Findings  []SecurityAuditFinding `json:"findings"`
}

// SecurityAuditSummary contains counts by severity
type SecurityAuditSummary struct {
	Critical int `json:"critical"`
	Warn     int `json:"warn"`
	Info     int `json:"info"`
}

// SecurityAuditFinding represents a single security finding
type SecurityAuditFinding struct {
	CheckID     string `json:"checkId"`
	Severity    string `json:"severity"` // "critical", "warn", "info"
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Remediation string `json:"remediation,omitempty"`
}
