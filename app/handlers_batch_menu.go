package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleBatchEditMenuView handles keyboard input in the batch edit menu view
func (m model) handleBatchEditMenuView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear batch selection and return to list view
		m.batch.selectedItems = make(map[int]bool)
		m.state = listView
		m.stateCursor = 0
		return m, nil

	case "up", "k":
		if m.stateCursor > 0 {
			m.stateCursor--
		}

	case "down", "j":
		maxOptions := 1 // Only State for now (0-indexed, so max is 0)
		if m.stateCursor < maxOptions {
			m.stateCursor++
		}

	case "ctrl+u", "pgup":
		// Jump up half page
		m.stateCursor = max(0, m.stateCursor-10)

	case "ctrl+d", "pgdown":
		// Jump down half page
		maxOptions := 1 // Only State for now
		m.stateCursor = min(maxOptions, m.stateCursor+10)

	case "enter":
		// Based on cursor position, determine which field to edit
		switch m.stateCursor {
		case 0: // State
			// Load states and show state picker (existing flow)
			if m.client != nil {
				m.loading = true
				m.statusMessage = "Loading states..."
				m.stateCursor = 0 // Reset for state picker
				return m, tea.Batch(
					loadWorkItemStates(m.client, "Task"),
					m.spinner.Tick,
				)
			}
			// Future cases:
			// case 1: // Sprint
			// case 2: // Assigned To
			// case 3: // Priority
		}
	}

	return m, nil
}
