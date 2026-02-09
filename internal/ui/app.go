package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazyclaw/lazyclaw/internal/config"
	"github.com/lazyclaw/lazyclaw/internal/gateway"
	"github.com/lazyclaw/lazyclaw/internal/models"
	"github.com/lazyclaw/lazyclaw/internal/state"
	"github.com/lazyclaw/lazyclaw/internal/ui/keys"
	"github.com/lazyclaw/lazyclaw/internal/ui/styles"
)

// AppMode represents the current mode of the application
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeHelp
	ModeSearch
	ModeActions
)

// FocusedPane represents which pane has focus
type FocusedPane int

const (
	PaneInstances FocusedPane = iota
	PaneDetails
)

// Tab represents the available detail tabs
type Tab int

const (
	TabOverview Tab = iota
	TabLogs
	TabHealth
	TabChannels
	TabAgents
	TabSessions
	TabEvents
	TabMemory
	TabSecurity
	TabSystem
)

func (t Tab) String() string {
	names := []string{"Overview", "Logs", "Health", "Channels", "Agents", "Sessions", "Events", "Memory", "Security", "System"}
	if int(t) < len(names) {
		return names[t]
	}
	return "Unknown"
}

// App is the main application model
type App struct {
	// Configuration
	config *config.Config

	// UI state
	mode             AppMode
	focusedPane      FocusedPane
	activeTab        Tab
	width            int
	height           int
	selectedInstance int // Currently selected instance index

	// Keys
	keys keys.KeyMap

	// Sub-models
	searchInput textinput.Model

	// Gateway connections - one per instance
	mockClient  *gateway.MockClient
	cliAdapters []*gateway.CLIAdapter // One adapter per configured instance

	// Current instance state
	connectionState  models.ConnectionState
	logs             []models.LogEvent
	healthSnapshot   *models.HealthSnapshot
	healthCheckResult *models.HealthCheckResult
	openclawStatus   *models.OpenClawStatus

	// Log streaming
	logChan       chan models.LogEvent
	logCtx        context.Context
	logCancel     context.CancelFunc
	logFollowing  bool // Whether log following is active

	// Flags
	logFollow bool
	mockMode  bool
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config, uiState *state.State, mockMode bool) *App {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	app := &App{
		config:      cfg,
		mode:        ModeNormal,
		focusedPane: FocusedPane(uiState.FocusedPane),
		activeTab:   Tab(uiState.ActiveTab),
		keys:        keys.DefaultKeyMap(),
		searchInput: ti,
		logFollow:   uiState.LogFollow,
		mockMode:    mockMode,
	}

	// Add a mock instance if in mock mode and no instances configured
	if mockMode && len(cfg.Instances) == 0 {
		cfg.Instances = append(cfg.Instances, models.InstanceProfile{
			Name:  "Mock Gateway",
			Mode:  models.ConnectionModeLocal,
		})
	}

	return app
}

// GetState returns the current UI state for persistence
func (a *App) GetState() *state.State {
	// Resolve selected instance index to name
	selectedName := ""
	if a.selectedInstance >= 0 && a.selectedInstance < len(a.config.Instances) {
		selectedName = a.config.Instances[a.selectedInstance].Name
	}

	return &state.State{
		SelectedInstance: selectedName,
		ActiveTab:        int(a.activeTab),
		FocusedPane:      int(a.focusedPane),
		LogFilter:        a.searchInput.Value(),
		LogFollow:        a.logFollow,
		WindowWidth:      a.width,
		WindowHeight:     a.height,
	}
}

// MockLogTickMsg is sent when a mock log event is available
type MockLogTickMsg struct{}

// CLIStatusMsg is sent when CLI status fetch completes
type CLIStatusMsg struct {
	Status *models.OpenClawStatus
	Error  error
}

// CLILogMsg is sent when a log event arrives from CLI
type CLILogMsg struct {
	Event models.LogEvent
}

// CLIHealthMsg is sent when CLI health fetch completes
type CLIHealthMsg struct {
	Result *models.HealthCheckResult
	Error  error
}

// RefreshTickMsg triggers periodic status refresh
type RefreshTickMsg struct{}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd

	if a.mockMode {
		// In mock mode, create mock client and start receiving logs
		cmds = append(cmds, a.connectMock())
	} else {
		// Create CLI adapters for all configured instances
		a.initCLIAdapters()

		// Fetch status and health for current instance
		cmds = append(cmds, a.fetchCLIStatus())
		cmds = append(cmds, a.fetchCLIHealth())

		// Start log following for current instance
		cmds = append(cmds, a.startLogFollowing())

		// Start periodic refresh
		cmds = append(cmds, a.scheduleRefresh())
	}

	return tea.Batch(cmds...)
}

// initCLIAdapters creates CLI adapters for all configured instances
func (a *App) initCLIAdapters() {
	a.cliAdapters = nil

	// If no instances configured, create a local adapter
	if len(a.config.Instances) == 0 {
		adapter := gateway.NewCLIAdapter()
		adapter.InstanceName = "Local"
		if a.config.OpenClawCLI != "" {
			adapter.BinaryPath = a.config.OpenClawCLI
		}
		a.cliAdapters = append(a.cliAdapters, adapter)
		return
	}

	// Create an adapter for each configured instance
	for _, inst := range a.config.Instances {
		var adapter *gateway.CLIAdapter

		switch inst.Mode {
		case models.ConnectionModeSSH:
			if inst.SSH != nil {
				// Check for openclaw_cli in both instance level and ssh level
				openclawPath := inst.OpenClawCLI
				if openclawPath == "" && inst.SSH.OpenClawCLI != "" {
					openclawPath = inst.SSH.OpenClawCLI
				}
				adapter = gateway.NewSSHCLIAdapter(inst.Name, inst.SSH, openclawPath)
			} else {
				// SSH mode but no SSH config - skip
				continue
			}
		default: // Local mode
			adapter = gateway.NewCLIAdapter()
			adapter.InstanceName = inst.Name
			if inst.OpenClawCLI != "" {
				adapter.BinaryPath = inst.OpenClawCLI
			} else if a.config.OpenClawCLI != "" {
				adapter.BinaryPath = a.config.OpenClawCLI
			}
		}

		a.cliAdapters = append(a.cliAdapters, adapter)
	}

	// Ensure we have at least one adapter
	if len(a.cliAdapters) == 0 {
		adapter := gateway.NewCLIAdapter()
		adapter.InstanceName = "Local"
		a.cliAdapters = append(a.cliAdapters, adapter)
	}
}

// getCurrentAdapter returns the CLI adapter for the currently selected instance
func (a *App) getCurrentAdapter() *gateway.CLIAdapter {
	if len(a.cliAdapters) == 0 {
		return nil
	}
	if a.selectedInstance < 0 || a.selectedInstance >= len(a.cliAdapters) {
		return a.cliAdapters[0]
	}
	return a.cliAdapters[a.selectedInstance]
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateViewportSizes()

	case tea.KeyMsg:
		// Handle help mode
		if a.mode == ModeHelp {
			if key.Matches(msg, a.keys.Escape) || key.Matches(msg, a.keys.Help) || msg.String() == "q" {
				a.mode = ModeNormal
				return a, nil
			}
			return a, nil
		}

		// Handle search mode
		if a.mode == ModeSearch {
			if key.Matches(msg, a.keys.Escape) {
				a.mode = ModeNormal
				a.searchInput.Reset()
				return a, nil
			}
			if key.Matches(msg, a.keys.Enter) {
				a.mode = ModeNormal
				// Filter is applied during rendering via a.searchInput.Value()
				return a, nil
			}
			var cmd tea.Cmd
			a.searchInput, cmd = a.searchInput.Update(msg)
			return a, cmd
		}

		// Normal mode keybindings
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			a.mode = ModeHelp
			return a, nil

		case key.Matches(msg, a.keys.Search):
			a.mode = ModeSearch
			a.searchInput.Focus()
			return a, textinput.Blink

		case key.Matches(msg, a.keys.Tab):
			if a.focusedPane == PaneInstances {
				a.focusedPane = PaneDetails
			} else {
				a.focusedPane = PaneInstances
			}

		case key.Matches(msg, a.keys.ShiftTab):
			if a.focusedPane == PaneDetails {
				a.focusedPane = PaneInstances
			} else {
				a.focusedPane = PaneDetails
			}

		case key.Matches(msg, a.keys.Tab1):
			a.activeTab = TabOverview
		case key.Matches(msg, a.keys.Tab2):
			a.activeTab = TabLogs
		case key.Matches(msg, a.keys.Tab3):
			a.activeTab = TabHealth
		case key.Matches(msg, a.keys.Tab4):
			a.activeTab = TabChannels
		case key.Matches(msg, a.keys.Tab5):
			a.activeTab = TabAgents
		case key.Matches(msg, a.keys.Tab6):
			a.activeTab = TabSessions
		case key.Matches(msg, a.keys.Tab7):
			a.activeTab = TabEvents
		case key.Matches(msg, a.keys.Tab8):
			a.activeTab = TabMemory
		case key.Matches(msg, a.keys.Tab9):
			a.activeTab = TabSecurity
		case key.Matches(msg, a.keys.Tab10):
			a.activeTab = TabSystem

		case key.Matches(msg, a.keys.ToggleFollow):
			a.logFollow = !a.logFollow

		case key.Matches(msg, a.keys.Reconnect):
			if a.mockMode {
				cmds = append(cmds, a.connectMock())
			} else if a.getCurrentAdapter() != nil {
				cmds = append(cmds, a.fetchCLIStatus())
				cmds = append(cmds, a.fetchCLIHealth())
				a.stopLogFollowing()
				cmds = append(cmds, a.startLogFollowing())
			}

		case key.Matches(msg, a.keys.Up):
			// Navigate instances when left pane is focused
			if a.focusedPane == PaneInstances && len(a.cliAdapters) > 1 {
				if a.selectedInstance > 0 {
					a.selectedInstance--
					a.switchInstance(&cmds)
				}
			}

		case key.Matches(msg, a.keys.Down):
			// Navigate instances when left pane is focused
			if a.focusedPane == PaneInstances && len(a.cliAdapters) > 1 {
				if a.selectedInstance < len(a.cliAdapters)-1 {
					a.selectedInstance++
					a.switchInstance(&cmds)
				}
			}

		case key.Matches(msg, a.keys.Enter):
			// Select instance and switch to details pane
			if a.focusedPane == PaneInstances {
				a.focusedPane = PaneDetails
				cmds = append(cmds, a.fetchCLIStatus())
				cmds = append(cmds, a.fetchCLIHealth())
			}
		}

	case gateway.ConnectedMsg:
		a.connectionState.Connected = true
		a.connectionState.LastError = ""
		a.connectionState.Scopes = msg.Scopes
		a.connectionState.ProtocolVersion = msg.ProtocolVersion
		a.connectionState.GatewayVersion = msg.GatewayVersion
		// If mock mode, start listening for mock logs
		if a.mockMode && a.mockClient != nil {
			cmds = append(cmds, a.waitForMockLog())
		}

	case gateway.DisconnectedMsg:
		a.connectionState.Connected = false
		a.connectionState.LastError = msg.Error

	case gateway.LogMsg:
		a.logs = append(a.logs, msg.Event)
		if len(a.logs) > a.config.UI.LogTailLines {
			a.logs = a.logs[1:]
		}
		// Continue listening for more logs in mock mode
		if a.mockMode && a.mockClient != nil {
			cmds = append(cmds, a.waitForMockLog())
		}

	case gateway.HealthMsg:
		a.healthSnapshot = &msg.Snapshot

	case CLIStatusMsg:
		if msg.Error != nil {
			a.connectionState.Connected = false
			a.connectionState.LastError = msg.Error.Error()
		} else {
			a.openclawStatus = msg.Status
			// Update connection state from CLI status
			if msg.Status.Gateway != nil {
				a.connectionState.Connected = msg.Status.Gateway.Reachable
				if msg.Status.Gateway.Self.Version != "" {
					a.connectionState.GatewayVersion = msg.Status.Gateway.Self.Version
				}
				if msg.Status.Gateway.Error != nil {
					a.connectionState.LastError = *msg.Status.Gateway.Error
				} else {
					a.connectionState.LastError = ""
				}
			}
		}

	case CLIHealthMsg:
		if msg.Error == nil {
			a.healthCheckResult = msg.Result
		}

	case CLILogMsg:
		a.logs = append(a.logs, msg.Event)
		if len(a.logs) > a.config.UI.LogTailLines {
			a.logs = a.logs[1:]
		}
		// Continue listening for more log events
		if a.logFollowing {
			cmds = append(cmds, a.waitForCLILog())
		}

	case RefreshTickMsg:
		// Refresh status periodically
		if !a.mockMode && a.getCurrentAdapter() != nil {
			cmds = append(cmds, a.fetchCLIStatus())
		}
		cmds = append(cmds, a.scheduleRefresh())

	}

	return a, tea.Batch(cmds...)
}

// View implements tea.Model
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	// Help overlay
	if a.mode == ModeHelp {
		return a.renderHelp()
	}

	// Main layout
	return a.renderMainLayout()
}

func (a *App) renderMainLayout() string {
	// Calculate dimensions
	leftWidth := 25
	rightWidth := a.width - leftWidth - 3 // Account for borders
	contentHeight := a.height - 4          // Account for bottom bar and borders

	// Render left pane (instances)
	leftPane := a.renderInstancesPane(leftWidth, contentHeight)

	// Render right pane (details with tabs)
	rightPane := a.renderDetailsPane(rightWidth, contentHeight)

	// Combine panes
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Bottom bar
	bottomBar := a.renderBottomBar()

	// Search bar if active
	if a.mode == ModeSearch {
		searchBar := a.renderSearchBar()
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, searchBar, bottomBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, bottomBar)
}

func (a *App) renderInstancesPane(width, height int) string {
	var style lipgloss.Style
	if a.focusedPane == PaneInstances {
		style = styles.FocusedPaneBorder
	} else {
		style = styles.PaneBorder
	}
	style = style.Width(width).Height(height)

	title := styles.TitleStyle.Render("Instances")

	var lines []string

	// Show adapters (which match configured instances or local)
	if len(a.cliAdapters) == 0 {
		lines = append(lines, styles.Muted.Render("Detecting gateway..."))
	} else {
		for i, adapter := range a.cliAdapters {
			// Get status badge for this adapter
			status := a.getAdapterStatusBadge(adapter)

			// Build instance line
			name := adapter.GetInstanceName()
			if name == "" {
				name = "Instance " + fmt.Sprintf("%d", i+1)
			}

			// Add mode indicator
			modeIndicator := ""
			if adapter.IsRemote() {
				modeIndicator = styles.Muted.Render(" [SSH]")
			}

			line := status + " " + name + modeIndicator

			if i == a.selectedInstance {
				lines = append(lines, styles.SelectedItem.Render(line))
			} else {
				lines = append(lines, styles.UnselectedItem.Render(line))
			}
		}
	}

	content := strings.Join(lines, "\n")
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// getAdapterStatusBadge returns a status badge for a specific adapter
func (a *App) getAdapterStatusBadge(adapter *gateway.CLIAdapter) string {
	if adapter == nil {
		return styles.StatusDegraded.Render("[...]")
	}

	// For the current adapter, use cached status
	if adapter == a.getCurrentAdapter() {
		if a.openclawStatus != nil && a.openclawStatus.Gateway != nil {
			if a.openclawStatus.Gateway.Reachable {
				return styles.StatusOK.Render("[OK]")
			}
			return styles.StatusDown.Render("[DOWN]")
		}
		if a.connectionState.LastError != "" {
			return styles.StatusDown.Render("[ERR]")
		}
	}

	// For other adapters, check their cached status
	cached := adapter.GetCachedStatus()
	if cached != nil && cached.Gateway != nil {
		if cached.Gateway.Reachable {
			return styles.StatusOK.Render("[OK]")
		}
		return styles.StatusDown.Render("[DOWN]")
	}

	if adapter.GetLastError() != nil {
		return styles.StatusDown.Render("[ERR]")
	}

	return styles.StatusDegraded.Render("[...]")
}

func (a *App) renderDetailsPane(width, height int) string {
	var style lipgloss.Style
	if a.focusedPane == PaneDetails {
		style = styles.FocusedPaneBorder
	} else {
		style = styles.PaneBorder
	}
	style = style.Width(width).Height(height)

	// Render tabs
	tabs := a.renderTabs()

	// Render tab content
	contentHeight := height - 3 // Account for tabs
	var content string
	switch a.activeTab {
	case TabOverview:
		content = a.renderOverviewTab(width-2, contentHeight)
	case TabLogs:
		content = a.renderLogsTab(width-2, contentHeight)
	case TabHealth:
		content = a.renderHealthTab(width-2, contentHeight)
	case TabChannels:
		content = a.renderChannelsTab(width-2, contentHeight)
	case TabAgents:
		content = a.renderAgentsTab(width-2, contentHeight)
	case TabSessions:
		content = a.renderSessionsTab(width-2, contentHeight)
	case TabEvents:
		content = a.renderEventsTab(width-2, contentHeight)
	case TabMemory:
		content = a.renderMemoryTab(width-2, contentHeight)
	case TabSecurity:
		content = a.renderSecurityTab(width-2, contentHeight)
	case TabSystem:
		content = a.renderSystemTab(width-2, contentHeight)
	default:
		content = styles.Muted.Render("Tab not implemented")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, tabs, content))
}

func (a *App) renderTabs() string {
	var tabs []string
	allTabs := []Tab{
		TabOverview, TabLogs, TabHealth, TabChannels, TabAgents,
		TabSessions, TabEvents, TabMemory, TabSecurity, TabSystem,
	}

	for _, t := range allTabs {
		if t == a.activeTab {
			tabs = append(tabs, styles.ActiveTab.Render(t.String()))
		} else {
			tabs = append(tabs, styles.InactiveTab.Render(t.String()))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (a *App) renderOverviewTab(width, height int) string {
	var lines []string

	// If we have real OpenClaw status, show that
	if a.openclawStatus != nil {
		return a.renderRealOverview(width, height)
	}

	// Fallback to basic connection info
	lines = append(lines, styles.HelpSection.Render("Connection"))

	if len(a.config.Instances) == 0 && !a.mockMode {
		lines = append(lines, styles.Muted.Render("No instance configured"))
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render("Checking openclaw CLI..."))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	if len(a.config.Instances) > 0 {
		inst := a.config.Instances[0]
		lines = append(lines, "  Name: "+inst.Name)
		lines = append(lines, "  Mode: "+string(inst.Mode))
		if inst.SSH != nil {
			lines = append(lines, "  Host: "+inst.SSH.Host)
		}
		lines = append(lines, "")
	}

	lines = append(lines, styles.HelpSection.Render("Status"))
	if a.connectionState.Connected {
		lines = append(lines, "  State:    "+styles.StatusOK.Render("CONNECTED"))
		lines = append(lines, "  Scopes:   "+formatScopes(a.connectionState.Scopes))
		if a.connectionState.ProtocolVersion != "" {
			lines = append(lines, "  Protocol: "+a.connectionState.ProtocolVersion)
		}
		if a.connectionState.GatewayVersion != "" {
			lines = append(lines, "  Gateway:  "+a.connectionState.GatewayVersion)
		}
	} else {
		lines = append(lines, "  State: "+styles.StatusDown.Render("DISCONNECTED"))
		if a.connectionState.LastError != "" {
			lines = append(lines, "  Error: "+styles.LogError.Render(a.connectionState.LastError))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (a *App) renderRealOverview(width, height int) string {
	var lines []string
	status := a.openclawStatus

	// Quick status summary at top
	lines = append(lines, styles.HelpSection.Render("Quick Status"))
	lines = append(lines, "")

	// Gateway status with latency
	if status.Gateway != nil {
		gw := status.Gateway
		if gw.Reachable {
			lines = append(lines, fmt.Sprintf("  Gateway:    %s (%dms latency)",
				styles.BadgeOK.Render("ONLINE"), gw.ConnectLatencyMs))
		} else {
			lines = append(lines, "  Gateway:    "+styles.BadgeError.Render("OFFLINE"))
		}
	}

	// Service status compact
	if status.GatewayService != nil && status.GatewayService.Installed {
		if contains(status.GatewayService.RuntimeShort, "running") {
			lines = append(lines, "  Service:    "+styles.BadgeOK.Render("RUNNING"))
		} else {
			lines = append(lines, "  Service:    "+styles.BadgeError.Render("STOPPED"))
		}
	}

	// Sessions count
	if status.Sessions != nil {
		lines = append(lines, fmt.Sprintf("  Sessions:   %s active",
			styles.LabelValueHighlight.Render(fmt.Sprintf("%d", status.Sessions.Count))))
	}

	// Agents count
	if status.Agents != nil {
		lines = append(lines, fmt.Sprintf("  Agents:     %d configured (default: %s)",
			len(status.Agents.Agents), status.Agents.DefaultID))
	}

	// Security summary with colored badges
	if status.SecurityAudit != nil {
		summary := status.SecurityAudit.Summary
		secLine := "  Security:   "
		if summary.Critical > 0 {
			secLine += styles.SeverityCritical.Render(fmt.Sprintf(" %d ", summary.Critical))
		}
		if summary.Warn > 0 {
			secLine += styles.SeverityWarn.Render(fmt.Sprintf(" %d ", summary.Warn))
		}
		if summary.Critical == 0 && summary.Warn == 0 {
			secLine += styles.BadgeOK.Render("OK")
		}
		lines = append(lines, secLine)
	}
	lines = append(lines, "")

	// Channels summary
	if len(status.ChannelSummary) > 0 {
		lines = append(lines, styles.HelpSection.Render("Channels"))
		for _, ch := range status.ChannelSummary {
			if ch != "" && ch[0] != ' ' {
				// Colorize based on status
				if contains(ch, "linked") {
					lines = append(lines, "  "+styles.StatusOK.Render("●")+" "+ch)
				} else if contains(ch, "configured") {
					lines = append(lines, "  "+styles.StatusOK.Render("●")+" "+ch)
				} else {
					lines = append(lines, "  "+styles.Muted.Render("○")+" "+ch)
				}
			}
		}
		lines = append(lines, "")
	}

	// Model & token info
	if status.Sessions != nil {
		lines = append(lines, styles.HelpSection.Render("Model Configuration"))
		lines = append(lines, fmt.Sprintf("  Model:   %s", styles.LabelValueHighlight.Render(status.Sessions.Defaults.Model)))
		lines = append(lines, fmt.Sprintf("  Context: %s tokens", formatNumber(status.Sessions.Defaults.ContextTokens)))
		lines = append(lines, "")
	}

	// Memory summary
	if status.Memory != nil {
		lines = append(lines, styles.HelpSection.Render("Memory (RAG)"))
		features := []string{}
		if status.Memory.Vector.Enabled && status.Memory.Vector.Available {
			features = append(features, "vector")
		}
		if status.Memory.FTS.Enabled && status.Memory.FTS.Available {
			features = append(features, "FTS")
		}
		if status.Memory.Cache.Enabled {
			features = append(features, "cache")
		}
		lines = append(lines, fmt.Sprintf("  %d files, %d chunks [%s]",
			status.Memory.Files, status.Memory.Chunks, strings.Join(features, ", ")))
		if status.Memory.Dirty {
			lines = append(lines, "  "+styles.LogWarn.Render("Index needs refresh"))
		}
		lines = append(lines, "")
	}

	// Recent activity from sessions
	if status.Sessions != nil && len(status.Sessions.Recent) > 0 {
		lines = append(lines, styles.HelpSection.Render("Recent Activity"))
		maxRecent := 5
		if len(status.Sessions.Recent) < maxRecent {
			maxRecent = len(status.Sessions.Recent)
		}
		for _, sess := range status.Sessions.Recent[:maxRecent] {
			age := formatAge(sess.Age)
			pct := sess.PercentUsed

			// Mini progress indicator
			var pctStyle lipgloss.Style
			if pct >= 80 {
				pctStyle = styles.LogError
			} else if pct >= 50 {
				pctStyle = styles.LogWarn
			} else {
				pctStyle = styles.Muted
			}

			lines = append(lines, fmt.Sprintf("  %s %s (%s ago) %s",
				styles.Muted.Render("●"),
				truncate(sess.Key, 40),
				age,
				pctStyle.Render(fmt.Sprintf("%d%%", pct))))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Sessions Tab
// ============================================================================

func (a *App) renderSessionsTab(width, height int) string {
	if a.openclawStatus == nil || a.openclawStatus.Sessions == nil {
		return styles.Muted.Render("No session data available")
	}

	sessions := a.openclawStatus.Sessions
	var lines []string

	// Summary header
	lines = append(lines, styles.HelpSection.Render("Session Summary"))
	lines = append(lines, fmt.Sprintf("  Total Sessions: %s", styles.LabelValueHighlight.Render(fmt.Sprintf("%d", sessions.Count))))
	lines = append(lines, fmt.Sprintf("  Default Model:  %s", sessions.Defaults.Model))
	lines = append(lines, fmt.Sprintf("  Context Window: %s tokens", formatNumber(sessions.Defaults.ContextTokens)))
	lines = append(lines, "")

	// Recent sessions table
	lines = append(lines, styles.HelpSection.Render("Recent Sessions"))
	lines = append(lines, "")

	// Table header
	header := fmt.Sprintf("  %-12s %-8s %-10s %8s %8s %6s", "Agent", "Kind", "Age", "Tokens", "Remain", "Used")
	lines = append(lines, styles.TableHeader.Render(header))

	// Show recent sessions with token usage bars
	maxSessions := height - 10
	if maxSessions > len(sessions.Recent) {
		maxSessions = len(sessions.Recent)
	}
	if maxSessions < 0 {
		maxSessions = 0
	}

	for i, sess := range sessions.Recent[:maxSessions] {
		age := formatAge(sess.Age)
		tokens := formatNumber(sess.TotalTokens)
		remain := formatNumber(sess.RemainingTokens)
		pct := fmt.Sprintf("%d%%", sess.PercentUsed)

		// Color based on usage
		pctStyle := styles.LabelValue
		if sess.PercentUsed >= 80 {
			pctStyle = styles.LogError
		} else if sess.PercentUsed >= 50 {
			pctStyle = styles.LogWarn
		}

		row := fmt.Sprintf("  %-12s %-8s %-10s %8s %8s %s",
			truncate(sess.AgentID, 12),
			sess.Kind,
			age,
			tokens,
			remain,
			pctStyle.Render(pct),
		)

		if i%2 == 0 {
			lines = append(lines, row)
		} else {
			lines = append(lines, styles.TableRowAlt.Render(row))
		}

		// Add progress bar
		bar := renderProgressBar(sess.PercentUsed, width-6)
		lines = append(lines, "    "+bar)
	}

	if len(sessions.Recent) > maxSessions {
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render(fmt.Sprintf("  ... and %d more sessions", len(sessions.Recent)-maxSessions)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Agents Tab
// ============================================================================

func (a *App) renderAgentsTab(width, height int) string {
	if a.openclawStatus == nil || a.openclawStatus.Agents == nil {
		return styles.Muted.Render("No agent data available")
	}

	agents := a.openclawStatus.Agents
	var lines []string

	// Summary
	lines = append(lines, styles.HelpSection.Render("Agent Summary"))
	lines = append(lines, fmt.Sprintf("  Default Agent:    %s", styles.LabelValueHighlight.Render(agents.DefaultID)))
	lines = append(lines, fmt.Sprintf("  Total Agents:     %d", len(agents.Agents)))
	lines = append(lines, fmt.Sprintf("  Total Sessions:   %d", agents.TotalSessions))
	if agents.BootstrapPendingCount > 0 {
		lines = append(lines, fmt.Sprintf("  Bootstrap Pending: %s", styles.LogWarn.Render(fmt.Sprintf("%d", agents.BootstrapPendingCount))))
	}
	lines = append(lines, "")

	// Agent details
	for _, agent := range agents.Agents {
		lines = append(lines, styles.HelpSection.Render(fmt.Sprintf("Agent: %s", agent.ID)))

		// Status badge
		if agent.BootstrapPending {
			lines = append(lines, "  Status:     "+styles.BadgeWarning.Render("BOOTSTRAP PENDING"))
		} else {
			lines = append(lines, "  Status:     "+styles.BadgeOK.Render("READY"))
		}

		lines = append(lines, fmt.Sprintf("  Workspace:  %s", truncatePath(agent.WorkspaceDir, width-14)))
		lines = append(lines, fmt.Sprintf("  Sessions:   %d", agent.SessionsCount))
		lines = append(lines, fmt.Sprintf("  Last Active: %s ago", formatAge(agent.LastActiveAgeMs)))
		lines = append(lines, "")
	}

	// Heartbeat info
	if a.openclawStatus.Heartbeat != nil {
		hb := a.openclawStatus.Heartbeat
		lines = append(lines, styles.HelpSection.Render("Heartbeat Configuration"))
		lines = append(lines, fmt.Sprintf("  Default Agent: %s", hb.DefaultAgentID))
		for _, agent := range hb.Agents {
			status := styles.Muted.Render("disabled")
			if agent.Enabled {
				status = styles.StatusOK.Render("enabled")
			}
			lines = append(lines, fmt.Sprintf("  - %s: %s (every %s)", agent.AgentID, status, agent.Every))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Channels Tab
// ============================================================================

func (a *App) renderChannelsTab(width, height int) string {
	if a.openclawStatus == nil {
		return styles.Muted.Render("No channel data available")
	}

	var lines []string

	lines = append(lines, styles.HelpSection.Render("Channel Status"))
	lines = append(lines, "")

	// Link channel (WhatsApp)
	if a.openclawStatus.LinkChannel != nil {
		lc := a.openclawStatus.LinkChannel
		lines = append(lines, styles.CardTitle.Render(fmt.Sprintf("  %s", lc.Label)))

		if lc.Linked {
			lines = append(lines, "    Status:   "+styles.BadgeOK.Render("LINKED"))
			authAge := formatAge(int64(lc.AuthAgeMs))
			lines = append(lines, fmt.Sprintf("    Auth Age: %s", authAge))
		} else {
			lines = append(lines, "    Status:   "+styles.BadgeError.Render("NOT LINKED"))
		}
		lines = append(lines, "")
	}

	// Channel summary from the status
	if len(a.openclawStatus.ChannelSummary) > 0 {
		lines = append(lines, styles.HelpSection.Render("Channel Configuration"))
		lines = append(lines, "")

		for _, ch := range a.openclawStatus.ChannelSummary {
			if ch == "" {
				continue
			}
			// Main channel lines start without space, details are indented
			if ch[0] == ' ' {
				lines = append(lines, styles.Muted.Render("  "+ch))
			} else {
				// Parse channel status from summary line
				if contains(ch, "linked") || contains(ch, "configured") {
					lines = append(lines, "  "+styles.StatusOK.Render("*")+" "+ch)
				} else {
					lines = append(lines, "  "+styles.Muted.Render("*")+" "+ch)
				}
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Memory Tab
// ============================================================================

func (a *App) renderMemoryTab(width, height int) string {
	if a.openclawStatus == nil || a.openclawStatus.Memory == nil {
		return styles.Muted.Render("No memory/RAG data available")
	}

	mem := a.openclawStatus.Memory
	var lines []string

	lines = append(lines, styles.HelpSection.Render("Memory System (RAG)"))
	lines = append(lines, "")

	// Overview
	lines = append(lines, "  "+styles.CardTitle.Render("Configuration"))
	lines = append(lines, fmt.Sprintf("    Backend:   %s", mem.Backend))
	lines = append(lines, fmt.Sprintf("    Agent:     %s", mem.AgentID))
	lines = append(lines, fmt.Sprintf("    Provider:  %s (%s)", mem.Provider, mem.Model))
	lines = append(lines, fmt.Sprintf("    Workspace: %s", truncatePath(mem.WorkspaceDir, width-16)))
	lines = append(lines, fmt.Sprintf("    Database:  %s", truncatePath(mem.DBPath, width-16)))
	lines = append(lines, "")

	// Content stats
	lines = append(lines, "  "+styles.CardTitle.Render("Content"))
	lines = append(lines, fmt.Sprintf("    Files:  %d", mem.Files))
	lines = append(lines, fmt.Sprintf("    Chunks: %d", mem.Chunks))
	if mem.Dirty {
		lines = append(lines, "    Status: "+styles.LogWarn.Render("DIRTY (needs reindex)"))
	} else {
		lines = append(lines, "    Status: "+styles.StatusOK.Render("CLEAN"))
	}
	lines = append(lines, "")

	// Source breakdown
	if len(mem.SourceCounts) > 0 {
		lines = append(lines, "  "+styles.CardTitle.Render("Sources"))
		for _, src := range mem.SourceCounts {
			lines = append(lines, fmt.Sprintf("    - %s: %d files, %d chunks", src.Source, src.Files, src.Chunks))
		}
		lines = append(lines, "")
	}

	// Features
	lines = append(lines, "  "+styles.CardTitle.Render("Features"))

	// Vector search
	if mem.Vector.Enabled {
		if mem.Vector.Available {
			lines = append(lines, fmt.Sprintf("    Vector Search: %s (%d dimensions)",
				styles.StatusOK.Render("enabled"), mem.Vector.Dims))
		} else {
			lines = append(lines, "    Vector Search: "+styles.LogWarn.Render("enabled but not available"))
		}
	} else {
		lines = append(lines, "    Vector Search: "+styles.Muted.Render("disabled"))
	}

	// FTS
	if mem.FTS.Enabled {
		if mem.FTS.Available {
			lines = append(lines, "    Full-Text Search: "+styles.StatusOK.Render("enabled"))
		} else {
			lines = append(lines, "    Full-Text Search: "+styles.LogWarn.Render("enabled but not available"))
		}
	} else {
		lines = append(lines, "    Full-Text Search: "+styles.Muted.Render("disabled"))
	}

	// Cache
	if mem.Cache.Enabled {
		lines = append(lines, fmt.Sprintf("    Embedding Cache: %s (%d entries)",
			styles.StatusOK.Render("enabled"), mem.Cache.Entries))
	} else {
		lines = append(lines, "    Embedding Cache: "+styles.Muted.Render("disabled"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Security Tab
// ============================================================================

func (a *App) renderSecurityTab(width, height int) string {
	if a.openclawStatus == nil || a.openclawStatus.SecurityAudit == nil {
		return styles.Muted.Render("No security audit data available")
	}

	audit := a.openclawStatus.SecurityAudit
	var lines []string

	lines = append(lines, styles.HelpSection.Render("Security Audit"))
	lines = append(lines, "")

	// Summary badges
	summary := audit.Summary
	summaryLine := "  "
	if summary.Critical > 0 {
		summaryLine += styles.SeverityCritical.Render(fmt.Sprintf(" %d CRITICAL ", summary.Critical)) + " "
	}
	if summary.Warn > 0 {
		summaryLine += styles.SeverityWarn.Render(fmt.Sprintf(" %d WARN ", summary.Warn)) + " "
	}
	if summary.Info > 0 {
		summaryLine += styles.SeverityInfo.Render(fmt.Sprintf(" %d INFO ", summary.Info))
	}
	lines = append(lines, summaryLine)
	lines = append(lines, "")

	// Findings
	lines = append(lines, styles.HelpSection.Render("Findings"))
	lines = append(lines, "")

	for _, finding := range audit.Findings {
		// Severity badge
		var severityBadge string
		switch finding.Severity {
		case "critical":
			severityBadge = styles.SeverityCritical.Render(" CRITICAL ")
		case "warn":
			severityBadge = styles.SeverityWarn.Render(" WARN ")
		default:
			severityBadge = styles.SeverityInfo.Render(" INFO ")
		}

		lines = append(lines, "  "+severityBadge+" "+styles.CardTitle.Render(finding.Title))

		// Detail (wrap if too long)
		detailLines := wrapText(finding.Detail, width-6)
		for _, dl := range detailLines {
			lines = append(lines, "    "+styles.Muted.Render(dl))
		}

		// Remediation
		if finding.Remediation != "" {
			lines = append(lines, "    "+styles.StatusOK.Render("Fix: ")+finding.Remediation)
		}
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// System Tab
// ============================================================================

func (a *App) renderSystemTab(width, height int) string {
	if a.openclawStatus == nil {
		return styles.Muted.Render("No system data available")
	}

	status := a.openclawStatus
	var lines []string

	// Gateway info
	if status.Gateway != nil {
		gw := status.Gateway
		lines = append(lines, styles.HelpSection.Render("Gateway"))
		if gw.Reachable {
			lines = append(lines, "  Status:  "+styles.BadgeOK.Render("REACHABLE"))
		} else {
			lines = append(lines, "  Status:  "+styles.BadgeError.Render("UNREACHABLE"))
		}
		lines = append(lines, fmt.Sprintf("  URL:     %s", gw.URL))
		lines = append(lines, fmt.Sprintf("  Mode:    %s", gw.Mode))
		lines = append(lines, fmt.Sprintf("  Source:  %s", gw.URLSource))
		lines = append(lines, fmt.Sprintf("  Latency: %dms", gw.ConnectLatencyMs))
		if gw.Self.Host != "" {
			lines = append(lines, fmt.Sprintf("  Host:    %s", gw.Self.Host))
			lines = append(lines, fmt.Sprintf("  IP:      %s", gw.Self.IP))
			lines = append(lines, fmt.Sprintf("  Version: %s", gw.Self.Version))
			lines = append(lines, fmt.Sprintf("  Platform: %s", gw.Self.Platform))
		}
		lines = append(lines, "")
	}

	// Services
	lines = append(lines, styles.HelpSection.Render("Services"))
	if status.GatewayService != nil {
		svc := status.GatewayService
		svcBadge := styles.BadgeMuted.Render("NOT INSTALLED")
		if svc.Installed {
			if contains(svc.RuntimeShort, "running") {
				svcBadge = styles.BadgeOK.Render("RUNNING")
			} else {
				svcBadge = styles.BadgeError.Render("STOPPED")
			}
		}
		lines = append(lines, "  Gateway Service: "+svcBadge)
		if svc.RuntimeShort != "" {
			lines = append(lines, fmt.Sprintf("    %s", styles.Muted.Render(svc.RuntimeShort)))
		}
	}
	if status.NodeService != nil {
		svc := status.NodeService
		svcBadge := styles.BadgeMuted.Render("NOT INSTALLED")
		if svc.Installed {
			if contains(svc.RuntimeShort, "running") {
				svcBadge = styles.BadgeOK.Render("RUNNING")
			} else {
				svcBadge = styles.BadgeError.Render("STOPPED")
			}
		}
		lines = append(lines, "  Node Service:    "+svcBadge)
	}
	lines = append(lines, "")

	// OS info
	if status.OS != nil {
		lines = append(lines, styles.HelpSection.Render("Operating System"))
		lines = append(lines, fmt.Sprintf("  Platform: %s", status.OS.Platform))
		lines = append(lines, fmt.Sprintf("  Arch:     %s", status.OS.Arch))
		lines = append(lines, fmt.Sprintf("  Release:  %s", status.OS.Release))
		lines = append(lines, "")
	}

	// Update info
	if status.Update != nil {
		lines = append(lines, styles.HelpSection.Render("Update Status"))
		lines = append(lines, fmt.Sprintf("  Install Kind: %s", status.Update.InstallKind))
		lines = append(lines, fmt.Sprintf("  Pkg Manager:  %s", status.Update.PackageManager))
		lines = append(lines, fmt.Sprintf("  Channel:      %s", status.UpdateChannel))
		if status.Update.Registry.LatestVersion != "" {
			lines = append(lines, fmt.Sprintf("  Latest:       %s", styles.LabelValueHighlight.Render(status.Update.Registry.LatestVersion)))
		}
		lines = append(lines, fmt.Sprintf("  Install Path: %s", truncatePath(status.Update.Root, width-16)))
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Logs Tab
// ============================================================================

func (a *App) renderLogsTab(width, height int) string {
	var lines []string

	// Header with follow status and filter info
	followBadge := styles.Muted.Render("[follow: off]")
	if a.logFollow {
		followBadge = styles.StatusOK.Render("[follow: on]")
	}
	filterInfo := ""
	if filter := a.searchInput.Value(); filter != "" {
		filterInfo = "  " + styles.Muted.Render("filter: ") + styles.LabelValueHighlight.Render(filter)
	}
	lines = append(lines, fmt.Sprintf("  %s  %s logs%s  %s",
		followBadge,
		styles.LabelValueHighlight.Render(fmt.Sprintf("%d", len(a.logs))),
		filterInfo,
		styles.Muted.Render("(f:follow /:search)")))
	lines = append(lines, "")

	if len(a.logs) == 0 {
		if a.logFollowing {
			lines = append(lines, styles.Muted.Render("  Waiting for log events..."))
		} else {
			lines = append(lines, styles.Muted.Render("  No logs available. Press r to reconnect."))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Filter logs
	filter := strings.ToLower(a.searchInput.Value())
	var filtered []models.LogEvent
	for _, log := range a.logs {
		if filter != "" && !strings.Contains(strings.ToLower(log.Message), filter) &&
			!strings.Contains(strings.ToLower(log.Level), filter) {
			continue
		}
		filtered = append(filtered, log)
	}

	// Calculate visible logs (show from the end if following)
	maxVisible := height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}

	startIdx := 0
	if a.logFollow && len(filtered) > maxVisible {
		startIdx = len(filtered) - maxVisible
	} else if len(filtered) > maxVisible {
		startIdx = len(filtered) - maxVisible
	}

	visible := filtered[startIdx:]
	if len(visible) > maxVisible {
		visible = visible[:maxVisible]
	}

	for _, log := range visible {
		var levelStyle lipgloss.Style
		var levelTag string
		switch log.Level {
		case "debug":
			levelStyle = styles.LogDebug
			levelTag = "DBG"
		case "warn", "warning":
			levelStyle = styles.LogWarn
			levelTag = "WRN"
		case "error":
			levelStyle = styles.LogError
			levelTag = "ERR"
		default:
			levelStyle = styles.LogInfo
			levelTag = "INF"
		}

		ts := log.Timestamp.Format("15:04:05")
		line := fmt.Sprintf("  %s %s %s",
			styles.Muted.Render(ts),
			levelStyle.Render(fmt.Sprintf("[%s]", levelTag)),
			levelStyle.Render(log.Message))
		lines = append(lines, line)
	}

	if filter != "" && len(filtered) != len(a.logs) {
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render(fmt.Sprintf("  Showing %d/%d logs (filtered)", len(filtered), len(a.logs))))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ============================================================================
// Health Tab
// ============================================================================

func (a *App) renderHealthTab(width, height int) string {
	var lines []string

	lines = append(lines, styles.HelpSection.Render("Gateway Health"))
	lines = append(lines, "")

	// If we have a health check result, display it
	if a.healthCheckResult != nil {
		return a.renderHealthCheckResult(width, height)
	}

	// Fall back to deriving health info from status
	if a.openclawStatus == nil {
		lines = append(lines, styles.Muted.Render("  No health data available. Waiting for health check..."))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Derive health level from status
	healthLevel := a.computeHealthLevel()
	switch healthLevel {
	case models.HealthOK:
		lines = append(lines, "  Overall: "+styles.BadgeOK.Render("OK"))
	case models.HealthDegraded:
		lines = append(lines, "  Overall: "+styles.BadgeWarning.Render("DEGRADED"))
	case models.HealthDown:
		lines = append(lines, "  Overall: "+styles.BadgeError.Render("DOWN"))
	}
	lines = append(lines, "")

	// Gateway reachability
	if a.openclawStatus.Gateway != nil {
		gw := a.openclawStatus.Gateway
		lines = append(lines, styles.HelpSection.Render("Gateway"))
		if gw.Reachable {
			lines = append(lines, fmt.Sprintf("  Reachable:  %s (%dms)",
				styles.StatusOK.Render("yes"), gw.ConnectLatencyMs))
		} else {
			lines = append(lines, "  Reachable:  "+styles.StatusDown.Render("no"))
			if gw.Error != nil {
				lines = append(lines, "  Error:      "+styles.LogError.Render(*gw.Error))
			}
		}
		lines = append(lines, "")
	}

	// Service health
	lines = append(lines, styles.HelpSection.Render("Services"))
	if a.openclawStatus.GatewayService != nil {
		svc := a.openclawStatus.GatewayService
		if svc.Installed {
			if contains(svc.RuntimeShort, "running") {
				lines = append(lines, "  Gateway Service: "+styles.StatusOK.Render("running"))
			} else {
				lines = append(lines, "  Gateway Service: "+styles.StatusDown.Render("stopped"))
			}
		} else {
			lines = append(lines, "  Gateway Service: "+styles.Muted.Render("not installed"))
		}
	}
	lines = append(lines, "")

	// Channel health derived from status
	if a.openclawStatus.LinkChannel != nil {
		lc := a.openclawStatus.LinkChannel
		lines = append(lines, styles.HelpSection.Render("Channel Health"))
		if lc.Linked {
			authAge := formatAge(int64(lc.AuthAgeMs))
			lines = append(lines, fmt.Sprintf("  %s: %s (auth: %s ago)",
				lc.Label, styles.StatusOK.Render("linked"), authAge))
		} else {
			lines = append(lines, fmt.Sprintf("  %s: %s",
				lc.Label, styles.StatusDown.Render("not linked")))
		}
		lines = append(lines, "")
	}

	// Security summary
	if a.openclawStatus.SecurityAudit != nil {
		summary := a.openclawStatus.SecurityAudit.Summary
		lines = append(lines, styles.HelpSection.Render("Security"))
		if summary.Critical > 0 {
			lines = append(lines, fmt.Sprintf("  %s critical findings",
				styles.SeverityCritical.Render(fmt.Sprintf("%d", summary.Critical))))
		}
		if summary.Warn > 0 {
			lines = append(lines, fmt.Sprintf("  %s warnings",
				styles.SeverityWarn.Render(fmt.Sprintf("%d", summary.Warn))))
		}
		if summary.Critical == 0 && summary.Warn == 0 {
			lines = append(lines, "  "+styles.StatusOK.Render("No issues found"))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (a *App) renderHealthCheckResult(width, height int) string {
	result := a.healthCheckResult
	var lines []string

	lines = append(lines, styles.HelpSection.Render("Gateway Health"))
	lines = append(lines, "")

	// Overall status badge
	switch strings.ToLower(result.Overall) {
	case "ok", "healthy", "pass":
		lines = append(lines, "  Overall: "+styles.BadgeOK.Render("OK"))
	case "degraded", "warning", "warn":
		lines = append(lines, "  Overall: "+styles.BadgeWarning.Render("DEGRADED"))
	case "down", "error", "fail", "unhealthy":
		lines = append(lines, "  Overall: "+styles.BadgeError.Render("DOWN"))
	default:
		lines = append(lines, "  Overall: "+styles.BadgeMuted.Render(strings.ToUpper(result.Overall)))
	}

	if result.ProbeDurationMs > 0 {
		lines = append(lines, fmt.Sprintf("  Probe Duration: %dms", result.ProbeDurationMs))
	}
	lines = append(lines, "")

	// Gateway health
	if result.Gateway != nil {
		gw := result.Gateway
		lines = append(lines, styles.HelpSection.Render("Gateway"))
		if gw.Reachable {
			lines = append(lines, fmt.Sprintf("  Reachable:  %s (%dms)",
				styles.StatusOK.Render("yes"), gw.LatencyMs))
		} else {
			lines = append(lines, "  Reachable:  "+styles.StatusDown.Render("no"))
			if gw.Error != "" {
				lines = append(lines, "  Error:      "+styles.LogError.Render(gw.Error))
			}
		}
		if gw.Version != "" {
			lines = append(lines, fmt.Sprintf("  Version:    %s", gw.Version))
		}
		lines = append(lines, "")
	}

	// Channel health items
	if len(result.Channels) > 0 {
		lines = append(lines, styles.HelpSection.Render("Channels"))
		for _, ch := range result.Channels {
			label := ch.Label
			if label == "" {
				label = ch.ID
			}
			switch strings.ToLower(ch.Status) {
			case "ok", "connected":
				lines = append(lines, fmt.Sprintf("  %s %s: %s",
					styles.StatusOK.Render("*"), label, styles.StatusOK.Render(ch.Status)))
			case "error", "fail":
				lines = append(lines, fmt.Sprintf("  %s %s: %s",
					styles.StatusDown.Render("*"), label, styles.StatusDown.Render(ch.Status)))
				if ch.Error != "" {
					lines = append(lines, "    "+styles.LogError.Render(ch.Error))
				}
			default:
				lines = append(lines, fmt.Sprintf("  %s %s: %s",
					styles.StatusDegraded.Render("*"), label, styles.Muted.Render(ch.Status)))
			}
		}
		lines = append(lines, "")
	}

	// Service health items
	if len(result.Services) > 0 {
		lines = append(lines, styles.HelpSection.Render("Services"))
		for _, svc := range result.Services {
			switch strings.ToLower(svc.Status) {
			case "running":
				lines = append(lines, fmt.Sprintf("  %s: %s",
					svc.Name, styles.StatusOK.Render("running")))
			case "stopped":
				lines = append(lines, fmt.Sprintf("  %s: %s",
					svc.Name, styles.StatusDown.Render("stopped")))
			default:
				lines = append(lines, fmt.Sprintf("  %s: %s",
					svc.Name, styles.Muted.Render(svc.Status)))
			}
			if svc.Details != "" {
				lines = append(lines, "    "+styles.Muted.Render(svc.Details))
			}
		}
		lines = append(lines, "")
	}

	// Doctor findings
	if len(result.Doctor) > 0 {
		lines = append(lines, styles.HelpSection.Render("Diagnostics"))
		for _, item := range result.Doctor {
			var statusBadge string
			switch strings.ToLower(item.Status) {
			case "pass", "ok":
				statusBadge = styles.StatusOK.Render("PASS")
			case "warn", "warning":
				statusBadge = styles.StatusDegraded.Render("WARN")
			case "fail", "error":
				statusBadge = styles.StatusDown.Render("FAIL")
			default:
				statusBadge = styles.Muted.Render(strings.ToUpper(item.Status))
			}
			lines = append(lines, fmt.Sprintf("  [%s] %s", statusBadge, item.Check))
			if item.Message != "" {
				lines = append(lines, "    "+styles.Muted.Render(item.Message))
			}
		}
		lines = append(lines, "")
	}

	// If raw output is available (JSON parse failed), show it
	if result.Raw != "" && result.Gateway == nil && len(result.Channels) == 0 {
		lines = append(lines, styles.HelpSection.Render("Raw Health Output"))
		lines = append(lines, "")
		rawLines := strings.Split(result.Raw, "\n")
		maxLines := height - 6
		if maxLines < 1 {
			maxLines = 1
		}
		for i, rl := range rawLines {
			if i >= maxLines {
				lines = append(lines, styles.Muted.Render(fmt.Sprintf("  ... %d more lines", len(rawLines)-maxLines)))
				break
			}
			lines = append(lines, "  "+styles.Muted.Render(rl))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// computeHealthLevel derives the health level from current status data
func (a *App) computeHealthLevel() models.HealthLevel {
	if a.openclawStatus == nil {
		return models.HealthDown
	}

	// Gateway unreachable = DOWN
	if a.openclawStatus.Gateway != nil && !a.openclawStatus.Gateway.Reachable {
		return models.HealthDown
	}

	// Check for degraded conditions
	if a.openclawStatus.LinkChannel != nil && !a.openclawStatus.LinkChannel.Linked {
		return models.HealthDegraded
	}
	if a.openclawStatus.SecurityAudit != nil && a.openclawStatus.SecurityAudit.Summary.Critical > 0 {
		return models.HealthDegraded
	}
	if a.openclawStatus.GatewayService != nil && a.openclawStatus.GatewayService.Installed &&
		!contains(a.openclawStatus.GatewayService.RuntimeShort, "running") {
		return models.HealthDegraded
	}

	return models.HealthOK
}

// ============================================================================
// Events Tab
// ============================================================================

// eventKeywords are used to filter logs into the events view
var eventKeywords = []string{
	"connect", "disconnect", "channel", "restart", "auth",
	"session", "error", "fail", "timeout", "linked", "unlinked",
	"start", "stop", "shutdown", "boot", "pair", "gateway",
}

func (a *App) renderEventsTab(width, height int) string {
	var lines []string

	lines = append(lines, styles.HelpSection.Render("System Events"))
	lines = append(lines, "")

	if len(a.logs) == 0 {
		lines = append(lines, styles.Muted.Render("  No events yet. Events are derived from the log stream."))
		if !a.logFollowing {
			lines = append(lines, styles.Muted.Render("  Press r to reconnect and start receiving logs."))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Filter logs to event-like entries
	var events []models.LogEvent
	for _, log := range a.logs {
		if isEventLog(log) {
			events = append(events, log)
		}
	}

	if len(events) == 0 {
		lines = append(lines, styles.Muted.Render("  No system events detected in log stream."))
		lines = append(lines, styles.Muted.Render(fmt.Sprintf("  (%d total log entries)", len(a.logs))))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	lines = append(lines, fmt.Sprintf("  %s events from %s log entries",
		styles.LabelValueHighlight.Render(fmt.Sprintf("%d", len(events))),
		styles.Muted.Render(fmt.Sprintf("%d", len(a.logs)))))
	lines = append(lines, "")

	// Show most recent events (from the end)
	maxVisible := height - 6
	if maxVisible < 1 {
		maxVisible = 1
	}

	startIdx := 0
	if len(events) > maxVisible {
		startIdx = len(events) - maxVisible
	}

	for _, event := range events[startIdx:] {
		var levelStyle lipgloss.Style
		var icon string
		switch event.Level {
		case "error":
			levelStyle = styles.LogError
			icon = styles.StatusDown.Render("!")
		case "warn", "warning":
			levelStyle = styles.LogWarn
			icon = styles.StatusDegraded.Render("*")
		default:
			levelStyle = styles.LogInfo
			icon = styles.StatusOK.Render("*")
		}

		ts := event.Timestamp.Format("15:04:05")
		line := fmt.Sprintf("  %s %s %s",
			styles.Muted.Render(ts),
			icon,
			levelStyle.Render(truncate(event.Message, width-16)))
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// isEventLog returns true if a log entry looks like a system event
func isEventLog(log models.LogEvent) bool {
	// All warn/error logs are events
	if log.Level == "warn" || log.Level == "warning" || log.Level == "error" {
		return true
	}

	// Check for event keywords in the message
	msgLower := strings.ToLower(log.Message)
	for _, kw := range eventKeywords {
		if strings.Contains(msgLower, kw) {
			return true
		}
	}

	return false
}

// ============================================================================
// Helper Functions
// ============================================================================

// Helper to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// formatAge converts milliseconds to human readable age
func formatAge(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// formatNumber formats large numbers with commas/k/M suffixes
func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// truncate truncates a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// truncatePath truncates a path, keeping the end visible
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen <= 6 {
		return path[len(path)-maxLen:]
	}
	return "..." + path[len(path)-maxLen+3:]
}

// wrapText wraps text to fit within maxWidth
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	words := splitWords(text)
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// splitWords splits text into words, handling newlines
func splitWords(text string) []string {
	var words []string
	current := ""
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// renderProgressBar renders a text-based progress bar
func renderProgressBar(percent int, width int) string {
	if width < 10 {
		width = 10
	}

	barWidth := width - 7 // Account for "[" + "]" + " XX%"
	if barWidth < 5 {
		barWidth = 5
	}

	filled := (percent * barWidth) / 100
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	// Choose color based on percentage
	var filledChar string
	if percent >= 80 {
		filledChar = styles.ProgressBarCritical.Render(strings.Repeat("█", filled))
	} else if percent >= 50 {
		filledChar = styles.ProgressBarWarning.Render(strings.Repeat("█", filled))
	} else {
		filledChar = styles.ProgressBarFilled.Render(strings.Repeat("█", filled))
	}

	emptyChar := styles.Muted.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("[%s%s] %3d%%", filledChar, emptyChar, percent)
}

func (a *App) renderBottomBar() string {
	hints := []string{
		styles.HintKey.Render("q") + styles.HintDesc.Render(":quit"),
		styles.HintKey.Render("?") + styles.HintDesc.Render(":help"),
		styles.HintKey.Render("1-0") + styles.HintDesc.Render(":tabs"),
		styles.HintKey.Render("/") + styles.HintDesc.Render(":search"),
		styles.HintKey.Render("f") + styles.HintDesc.Render(":follow"),
		styles.HintKey.Render("r") + styles.HintDesc.Render(":refresh"),
	}

	return styles.BottomBar.Width(a.width).Render(lipgloss.JoinHorizontal(lipgloss.Left, joinWithSeparator(hints, "  ")...))
}

func (a *App) renderSearchBar() string {
	prompt := styles.InputPrompt.Render("Search: ")
	return prompt + a.searchInput.View()
}

func (a *App) renderHelp() string {
	help := styles.HelpTitle.Render("lazyclaw Help") + "\n\n"

	help += styles.HelpSection.Render("Navigation") + "\n"
	help += "  tab/shift+tab  Switch between panes\n"
	help += "  j/k or arrows  Navigate lists\n"
	help += "  esc            Close modal/cancel\n\n"

	help += styles.HelpSection.Render("Tabs") + "\n"
	help += "  1  Overview    - Quick status summary\n"
	help += "  2  Logs        - Live log stream\n"
	help += "  3  Health      - Gateway health snapshot\n"
	help += "  4  Channels    - WhatsApp, Telegram status\n"
	help += "  5  Agents      - Agent configuration\n"
	help += "  6  Sessions    - Active sessions & token usage\n"
	help += "  7  Events      - System events feed\n"
	help += "  8  Memory      - RAG/vector search info\n"
	help += "  9  Security    - Security audit findings\n"
	help += "  0  System      - Services, OS, updates\n\n"

	help += styles.HelpSection.Render("Actions") + "\n"
	help += "  /              Search/filter logs\n"
	help += "  f              Toggle log follow mode\n"
	help += "  r              Refresh status\n"
	help += "  ?              Show this help\n"
	help += "  q              Quit\n\n"

	help += styles.Muted.Render("Press esc or ? to close")

	// Center the help overlay
	overlay := styles.HelpOverlay.Render(help)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (a *App) getStatusBadge() string {
	// Check OpenClaw status first
	if a.openclawStatus != nil && a.openclawStatus.Gateway != nil {
		if a.openclawStatus.Gateway.Reachable {
			return styles.StatusOK.Render("[OK]")
		}
		return styles.StatusDown.Render("[DOWN]")
	}

	if !a.connectionState.Connected {
		if a.connectionState.LastError != "" {
			return styles.StatusDown.Render("[DOWN]")
		}
		return styles.StatusDegraded.Render("[...]")
	}
	return styles.StatusOK.Render("[OK]")
}

func (a *App) updateViewportSizes() {
	// Update viewport sizes based on window dimensions
	// Currently a no-op as we render logs inline
}

func (a *App) connectMock() tea.Cmd {
	return func() tea.Msg {
		a.mockClient = gateway.NewMockClient()
		return a.mockClient.Connect()
	}
}

func (a *App) waitForMockLog() tea.Cmd {
	return func() tea.Msg {
		if a.mockClient == nil {
			return nil
		}
		log, ok := <-a.mockClient.GetLogs()
		if !ok {
			return gateway.DisconnectedMsg{Error: "mock client closed"}
		}
		return gateway.LogMsg{Event: log}
	}
}

func (a *App) fetchCLIStatus() tea.Cmd {
	return func() tea.Msg {
		adapter := a.getCurrentAdapter()
		if adapter == nil {
			return CLIStatusMsg{Error: fmt.Errorf("CLI adapter not initialized")}
		}
		status, err := adapter.GetFullStatus()
		return CLIStatusMsg{Status: status, Error: err}
	}
}

func (a *App) fetchCLIHealth() tea.Cmd {
	return func() tea.Msg {
		adapter := a.getCurrentAdapter()
		if adapter == nil {
			return CLIHealthMsg{Error: fmt.Errorf("CLI adapter not initialized")}
		}
		result, err := adapter.GetHealthSnapshot()
		return CLIHealthMsg{Result: result, Error: err}
	}
}

// startLogFollowing starts the log following process for the current adapter
func (a *App) startLogFollowing() tea.Cmd {
	return func() tea.Msg {
		adapter := a.getCurrentAdapter()
		if adapter == nil {
			return nil
		}

		// Create channel and context for log streaming
		a.logChan = make(chan models.LogEvent, 100)
		a.logCtx, a.logCancel = context.WithCancel(context.Background())

		if err := adapter.FollowLogs(a.logCtx, a.logChan); err != nil {
			// Log following failed to start - not fatal
			return CLILogMsg{Event: models.LogEvent{
				Timestamp: time.Now(),
				Level:     "warn",
				Source:    "lazyclaw",
				Message:   fmt.Sprintf("Could not start log following: %v", err),
			}}
		}

		a.logFollowing = true

		// Wait for the first log event
		select {
		case event, ok := <-a.logChan:
			if !ok {
				a.logFollowing = false
				return nil
			}
			return CLILogMsg{Event: event}
		case <-a.logCtx.Done():
			return nil
		}
	}
}

// waitForCLILog waits for the next log event from the CLI log channel
func (a *App) waitForCLILog() tea.Cmd {
	return func() tea.Msg {
		if a.logChan == nil {
			return nil
		}
		select {
		case event, ok := <-a.logChan:
			if !ok {
				a.logFollowing = false
				return nil
			}
			return CLILogMsg{Event: event}
		case <-a.logCtx.Done():
			return nil
		}
	}
}

// stopLogFollowing stops the current log following process
func (a *App) stopLogFollowing() {
	if a.logCancel != nil {
		a.logCancel()
	}
	a.logFollowing = false
	// Drain the channel
	if a.logChan != nil {
		go func() {
			for range a.logChan {
			}
		}()
	}
}

// switchInstance handles switching to a new instance
func (a *App) switchInstance(cmds *[]tea.Cmd) {
	a.openclawStatus = nil
	a.healthCheckResult = nil
	a.logs = nil
	a.stopLogFollowing()
	*cmds = append(*cmds, a.fetchCLIStatus())
	*cmds = append(*cmds, a.fetchCLIHealth())
	*cmds = append(*cmds, a.startLogFollowing())
}

func (a *App) scheduleRefresh() tea.Cmd {
	refreshMs := a.config.UI.RefreshMs
	if refreshMs <= 0 {
		refreshMs = 1000
	}
	return tea.Tick(time.Duration(refreshMs)*time.Millisecond, func(t time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

// Helper functions
func formatScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "none"
	}
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func joinWithSeparator(items []string, sep string) []string {
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, item)
	}
	return result
}
