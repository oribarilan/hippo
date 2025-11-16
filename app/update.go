package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store window dimensions
		m.width = msg.Width
		m.height = msg.Height

		// Initialize viewport when we know the window size
		if !m.viewportReady {
			m.viewport = viewport.New(msg.Width, msg.Height-5) // Reserve space for header/footer
			m.viewport.YPosition = 0
			m.viewportReady = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
		}

	case tea.KeyMsg:
		// Route keyboard input to appropriate handler based on current state
		switch m.state {
		case helpView:
			return m.handleHelpView(msg)
		case errorView:
			return m.handleErrorView(msg)
		case filterView:
			return m.handleFilterView(msg)
		case findView:
			return m.handleFindView(msg)
		case statePickerView:
			return m.handleStatePickerView(msg)
		case editView:
			return m.handleEditView(msg)
		case createView:
			return m.handleCreateView(msg)
		case deleteConfirmView:
			return m.handleDeleteConfirmView(msg)
		case listView:
			// Try global hotkeys first
			newModel, cmd, handled := m.handleGlobalHotkeys(msg)
			if handled {
				return newModel, cmd
			}
			// Then handle list-specific navigation
			return m.handleListViewNav(msg)
		case detailView:
			// Try global hotkeys first
			newModel, cmd, handled := m.handleGlobalHotkeys(msg)
			if handled {
				return newModel, cmd
			}
			// Then handle detail-specific navigation
			return m.handleDetailViewNav(msg)
		}

	case tasksLoadedMsg:
		return m.handleTasksLoadedMsg(msg)

	case stateUpdatedMsg:
		return m.handleStateUpdatedMsg(msg)

	case workItemUpdatedMsg:
		return m.handleWorkItemUpdatedMsg(msg)

	case statesLoadedMsg:
		return m.handleStatesLoadedMsg(msg)

	case workItemRefreshedMsg:
		return m.handleWorkItemRefreshedMsg(msg)

	case workItemCreatedMsg:
		return m.handleWorkItemCreatedMsg(msg)

	case workItemDeletedMsg:
		return m.handleWorkItemDeletedMsg(msg)

	case sprintsLoadedMsg:
		return m.handleSprintsLoadedMsg(msg)

	case spinner.TickMsg:
		if m.loading || m.loadingMore {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
