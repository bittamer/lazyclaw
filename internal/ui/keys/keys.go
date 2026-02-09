package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application
type KeyMap struct {
	Quit       key.Binding
	Help       key.Binding
	Search     key.Binding
	Tab        key.Binding
	ShiftTab   key.Binding
	Enter      key.Binding
	Escape     key.Binding
	Actions    key.Binding
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
	Tab5       key.Binding
	Tab6       key.Binding
	Tab7       key.Binding
	ToggleFollow key.Binding
	OpenConfig key.Binding
	EditConfig key.Binding
	Reconnect  key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/close"),
		),
		Actions: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "actions"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j/down", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdown", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "bottom"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "Overview"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "Logs"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "Health"),
		),
		Tab4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "Channels"),
		),
		Tab5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "Agents"),
		),
		Tab6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "Sessions"),
		),
		Tab7: key.NewBinding(
			key.WithKeys("7"),
			key.WithHelp("7", "Events"),
		),
		ToggleFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle follow"),
		),
		OpenConfig: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open config dir"),
		),
		EditConfig: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit config"),
		),
		Reconnect: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reconnect"),
		),
	}
}

// ShortHelp returns keybindings for the short help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help, k.Search, k.Tab}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Tab, k.ShiftTab, k.Enter, k.Escape},
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.Search, k.Actions, k.Help, k.Quit},
	}
}
