package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func initialModel(config *Config, configSource *ConfigSource) model {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Filter state inputs
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

	// Create mode input
	createInput := textinput.New()
	createInput.Placeholder = "Enter task title..."
	createInput.CharLimit = 255
	createInput.Width = 80

	return model{
		// Configuration
		config:       config,
		configSource: configSource,

		// Initialize WorkItemList maps
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),

		// Core state fields
		state:             loadingView,
		loading:           true,
		spinner:           s,
		availableStates:   []string{"New", "Active", "Closed", "Removed"},
		stateCategories:   make(map[string]string),
		currentMode:       sprintMode,
		currentTab:        currentSprint,
		currentBacklogTab: recentBacklog,
		sprints:           make(map[sprintTab]*Sprint),
		styles:            NewStyles(),

		// Grouped state initialization
		ui: UIState{
			cursor:       0,
			scrollOffset: 0,
		},
		edit: EditState{
			titleInput:       editTitleInput,
			descriptionInput: editDescriptionInput,
			fieldCursor:      0,
			fieldCount:       2, // Title and Description only
		},
		create: CreateState{
			input: createInput,
		},
		batch: BatchState{
			selectedItems: make(map[int]bool),
		},
		filter: FilterState{
			filteredTasks: []WorkItem{},
			filterInput:   filterInput,
			findInput:     findInput,
		},
	}
}

func (m model) Init() tea.Cmd {
	// If we're starting in wizard view, just return textinput blink command
	if m.state == configWizardView {
		return textinput.Blink
	}

	// Otherwise, initialize client and load data
	client, err := NewAzureDevOpsClient(m.config)
	if err != nil {
		return func() tea.Msg {
			return tasksLoadedMsg{err: err}
		}
	}
	// Only load sprint info initially, then load tasks for each sprint
	return tea.Batch(loadSprints(client), m.spinner.Tick)
}

// initialModelWithWizard creates a model that starts in the config wizard view
func initialModelWithWizard(existingConfig *Config, existingConfigSource *ConfigSource) model {
	// Create wizard input fields
	orgInput := textinput.New()
	orgInput.Placeholder = "https://dev.azure.com/your-org"
	orgInput.Focus()
	orgInput.CharLimit = 200
	orgInput.Width = 60
	if existingConfig != nil && existingConfig.OrganizationURL != "" {
		orgInput.SetValue(existingConfig.OrganizationURL)
	}

	projectInput := textinput.New()
	projectInput.Placeholder = "MyProject"
	projectInput.CharLimit = 100
	projectInput.Width = 60
	if existingConfig != nil && existingConfig.Project != "" {
		projectInput.SetValue(existingConfig.Project)
	}

	teamInput := textinput.New()
	teamInput.Placeholder = "MyTeam (optional, defaults to project name)"
	teamInput.CharLimit = 100
	teamInput.Width = 60
	if existingConfig != nil && existingConfig.Team != "" {
		teamInput.SetValue(existingConfig.Team)
	}

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		// Configuration
		config:       existingConfig,
		configSource: existingConfigSource,

		// Initialize WorkItemList maps
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),

		// Core state fields - start with wizard view
		state:             configWizardView,
		loading:           false,
		spinner:           s,
		availableStates:   []string{},
		stateCategories:   make(map[string]string),
		currentMode:       sprintMode,
		currentTab:        currentSprint,
		currentBacklogTab: recentBacklog,
		sprints:           make(map[sprintTab]*Sprint),
		styles:            NewStyles(),

		// Grouped state initialization
		ui: UIState{
			cursor:       0,
			scrollOffset: 0,
		},
		batch: BatchState{
			selectedItems: make(map[int]bool),
		},
		wizard: WizardState{
			fieldCursor:  0,
			orgInput:     orgInput,
			projectInput: projectInput,
			teamInput:    teamInput,
			err:          "",
		},
	}
}
