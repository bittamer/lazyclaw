package views

import (
	"github.com/lazyclaw/lazyclaw/internal/models"
)

// OverviewView displays the overview tab content
type OverviewView struct {
	width  int
	height int

	instance        *models.InstanceProfile
	connectionState *models.ConnectionState
	healthSnapshot  *models.HealthSnapshot
}

// NewOverviewView creates a new overview view
func NewOverviewView() *OverviewView {
	return &OverviewView{}
}

// SetSize sets the view dimensions
func (v *OverviewView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetData updates the view data
func (v *OverviewView) SetData(instance *models.InstanceProfile, state *models.ConnectionState, health *models.HealthSnapshot) {
	v.instance = instance
	v.connectionState = state
	v.healthSnapshot = health
}
