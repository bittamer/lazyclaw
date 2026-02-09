package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lazyclaw/lazyclaw/internal/config"
	"github.com/lazyclaw/lazyclaw/internal/state"
	"github.com/lazyclaw/lazyclaw/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse flags
	mockMode := flag.Bool("mock", false, "Run in mock mode (simulated data for UI testing)")
	flag.Parse()

	// Load or create configuration
	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load UI state
	uiState, _ := state.Load() // Ignore error, use defaults

	// Initialize the TUI application
	app := ui.NewApp(cfg, uiState, *mockMode)

	// Run the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazyclaw: %v\n", err)
		os.Exit(1)
	}

	// Save state on exit
	if finalApp, ok := finalModel.(*ui.App); ok {
		if saveState := finalApp.GetState(); saveState != nil {
			_ = state.Save(saveState) // Best effort save
		}
	}
}
