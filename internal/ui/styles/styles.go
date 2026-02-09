package styles

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary        = lipgloss.Color("#5DADE2")
	ColorSecondary      = lipgloss.Color("#82E0AA")
	ColorWarning        = lipgloss.Color("#F4D03F")
	ColorError          = lipgloss.Color("#E74C3C")
	ColorMuted          = lipgloss.Color("#7F8C8D")
	ColorBackground     = lipgloss.Color("#1C2833")
	ColorForeground     = lipgloss.Color("#ECF0F1")
	ColorHealthOK       = lipgloss.Color("#2ECC71")
	ColorHealthDegraded = lipgloss.Color("#F39C12")
	ColorHealthDown     = lipgloss.Color("#E74C3C")
	ColorDarkBg         = lipgloss.Color("#2C3E50")
)

// Text Styles
var (
	// Muted text style
	Muted = lipgloss.NewStyle().Foreground(ColorMuted)

	// Secondary text style
	Secondary = lipgloss.NewStyle().Foreground(ColorSecondary)

	// Primary text style
	Primary = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Base styles
	BaseStyle = lipgloss.NewStyle().Foreground(ColorForeground)
)

// Pane styles
var (
	PaneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)

	FocusedPaneBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)
)

// Title styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)
)

// Tab styles
var (
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorDarkBg).
			Padding(0, 2)

	InactiveTab = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 2)
)

// Status badge styles
var (
	StatusOK = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHealthOK)

	StatusDegraded = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHealthDegraded)

	StatusDown = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHealthDown)
)

// Bottom bar styles
var (
	BottomBar = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorDarkBg).
			Padding(0, 1)

	HintKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	HintDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Log level styles
var (
	LogDebug = lipgloss.NewStyle().Foreground(ColorMuted)
	LogInfo  = lipgloss.NewStyle().Foreground(ColorForeground)
	LogWarn  = lipgloss.NewStyle().Foreground(ColorWarning)
	LogError = lipgloss.NewStyle().Foreground(ColorError)
)

// Input styles
var (
	InputPrompt = lipgloss.NewStyle().Foreground(ColorPrimary)
)

// Help overlay styles
var (
	HelpOverlay = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	HelpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	HelpSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			MarginTop(1)
)

// Instance list styles
var (
	SelectedItem = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorDarkBg)

	UnselectedItem = lipgloss.NewStyle().
			Foreground(ColorForeground)
)

// Table/List styles
var (
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted)

	TableRow = lipgloss.NewStyle().
			Foreground(ColorForeground)

	TableRowAlt = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Background(lipgloss.Color("#1A252F"))

	TableRowSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorDarkBg)
)

// Progress bar styles
var (
	ProgressBarFilled = lipgloss.NewStyle().
				Background(ColorPrimary)

	ProgressBarEmpty = lipgloss.NewStyle().
				Background(ColorMuted)

	ProgressBarCritical = lipgloss.NewStyle().
				Background(ColorError)

	ProgressBarWarning = lipgloss.NewStyle().
				Background(ColorWarning)
)

// Card/Panel styles
var (
	Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	CardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	CardHighlight = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)
)

// Severity styles for security audit
var (
	SeverityCritical = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorError).
				Padding(0, 1)

	SeverityWarn = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorWarning).
			Padding(0, 1)

	SeverityInfo = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Padding(0, 1)
)

// Badge styles
var (
	BadgeOK = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorHealthOK).
		Padding(0, 1)

	BadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorHealthDegraded).
			Padding(0, 1)

	BadgeError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorHealthDown).
			Padding(0, 1)

	BadgeMuted = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Background(ColorMuted).
			Padding(0, 1)
)

// Label styles
var (
	LabelKey = lipgloss.NewStyle().
			Foreground(ColorMuted)

	LabelValue = lipgloss.NewStyle().
			Foreground(ColorForeground)

	LabelValueHighlight = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)
)

// Divider
var (
	Divider = lipgloss.NewStyle().
		Foreground(ColorMuted)
)
