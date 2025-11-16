package main

import (
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	filterInput := textinput.New()
	filterInput.Placeholder = "Filter by title or ID..."
	filterInput.Focus()

	findInput := textinput.New()
	findInput.Placeholder = "Find items (search in title/description)..."
	findInput.Focus()

	// Edit mode inputs
	editTitleInput := textinput.New()
	editTitleInput.Placeholder = "Title"
	editTitleInput.CharLimit = 255
	editTitleInput.Width = 80

	editDescriptionInput := textarea.New()
	editDescriptionInput.Placeholder = "Description"
	editDescriptionInput.CharLimit = 4000
	editDescriptionInput.SetWidth(80)
	editDescriptionInput.SetHeight(10)

	createInput := textinput.New()
	createInput.Placeholder = "Enter task title..."
	createInput.CharLimit = 255
	createInput.Width = 80

	return model{
		// Initialize WorkItemList maps
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),

		// UI state fields
		filteredTasks:        []WorkItem{},
		cursor:               0,
		scrollOffset:         0,
		state:                loadingView,
		loading:              true,
		spinner:              s,
		filterInput:          filterInput,
		findInput:            findInput,
		availableStates:      []string{"New", "Active", "Closed", "Removed"},
		stateCategories:      make(map[string]string),
		organizationURL:      os.Getenv("AZURE_DEVOPS_ORG_URL"),
		projectName:          os.Getenv("AZURE_DEVOPS_PROJECT"),
		currentMode:          sprintMode,
		currentTab:           currentSprint,
		currentBacklogTab:    recentBacklog,
		sprints:              make(map[sprintTab]*Sprint),
		editTitleInput:       editTitleInput,
		editDescriptionInput: editDescriptionInput,
		editFieldCursor:      0,
		editFieldCount:       2, // Title and Description only
		createInput:          createInput,
		selectedItems:        make(map[int]bool),
		styles:               NewStyles(),
	}
}

func (m model) Init() tea.Cmd {
	client, err := NewAzureDevOpsClient()
	if err != nil {
		return func() tea.Msg {
			return tasksLoadedMsg{err: err}
		}
	}
	// Only load sprint info initially, then load tasks for each sprint
	return tea.Batch(loadSprints(client), m.spinner.Tick)
}
