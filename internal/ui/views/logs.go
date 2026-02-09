package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/lazyclaw/lazyclaw/internal/models"
	"github.com/lazyclaw/lazyclaw/internal/ui/styles"
)

// LogsView displays and manages the logs tab
type LogsView struct {
	viewport viewport.Model
	logs     []models.LogEvent
	filter   string
	follow   bool
	width    int
	height   int
}

// NewLogsView creates a new logs view
func NewLogsView(width, height int) *LogsView {
	vp := viewport.New(width, height)
	return &LogsView{
		viewport: vp,
		logs:     make([]models.LogEvent, 0),
		follow:   true,
		width:    width,
		height:   height,
	}
}

// SetSize updates the view dimensions
func (v *LogsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
	v.updateContent()
}

// AddLog adds a log entry
func (v *LogsView) AddLog(log models.LogEvent) {
	v.logs = append(v.logs, log)
	v.updateContent()
}

// SetFilter sets the search filter
func (v *LogsView) SetFilter(filter string) {
	v.filter = filter
	v.updateContent()
}

// ClearFilter clears the search filter
func (v *LogsView) ClearFilter() {
	v.filter = ""
	v.updateContent()
}

// ScrollToBottom scrolls to the bottom of the log
func (v *LogsView) ScrollToBottom() {
	v.viewport.GotoBottom()
}

// ToggleFollow toggles follow mode
func (v *LogsView) ToggleFollow() {
	v.follow = !v.follow
}

// IsFollowing returns whether follow mode is enabled
func (v *LogsView) IsFollowing() bool {
	return v.follow
}

func (v *LogsView) updateContent() {
	var lines []string

	for _, log := range v.logs {
		// Apply filter if set
		if v.filter != "" && !strings.Contains(strings.ToLower(log.Message), strings.ToLower(v.filter)) {
			continue
		}

		var levelStyle = styles.LogInfo
		switch log.Level {
		case "debug":
			levelStyle = styles.LogDebug
		case "warn", "warning":
			levelStyle = styles.LogWarn
		case "error":
			levelStyle = styles.LogError
		}

		ts := log.Timestamp.Format("15:04:05")
		line := styles.Muted.Render(ts) + " " + levelStyle.Render(log.Message)
		lines = append(lines, line)
	}

	v.viewport.SetContent(strings.Join(lines, "\n"))
}

// View renders the logs view
func (v *LogsView) View() string {
	return v.viewport.View()
}
