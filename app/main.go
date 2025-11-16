package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

const version = "v0.1.0"

// defaultLoadLimit is the number of items to load per request
const defaultLoadLimit = 40

type viewState int

const (
	listView viewState = iota
	detailView
	statePickerView
	findView
	filterView
	helpView
	editView
	createView
	errorView
	deleteConfirmView
)

type appMode int

const (
	sprintMode appMode = iota
	backlogMode
)

type sprintTab int

const (
	previousSprint sprintTab = iota
	currentSprint
	nextSprint
)

type backlogTab int

const (
	recentBacklog backlogTab = iota
	abandonedWork
)

type Sprint struct {
	Name      string
	Path      string
	StartDate string
	EndDate   string
}

type model struct {
	// WorkItemList instances - component-based architecture
	sprintLists  map[sprintTab]*WorkItemList
	backlogLists map[backlogTab]*WorkItemList

	// Temporary filter state for backward compatibility
	filteredTasks []WorkItem
	filterActive  bool

	// Core UI state
	state             viewState
	selectedTask      *WorkItem
	selectedTaskID    int // Track which task the viewport is showing
	loading           bool
	loadingMore       bool // Track if we're loading more items (for inline spinner)
	err               error
	client            *AzureDevOpsClient
	spinner           spinner.Model
	filterInput       textinput.Model
	findInput         textinput.Model
	viewport          viewport.Model
	viewportReady     bool
	stateCursor       int
	availableStates   []string
	stateCategories   map[string]string // Map of state name to category (Proposed, InProgress, Completed, etc.)
	organizationURL   string
	projectName       string
	statusMessage     string
	lastActionLog     string    // Log line showing the result of the last action
	lastActionTime    time.Time // Timestamp of the last action
	currentMode       appMode
	currentTab        sprintTab
	currentBacklogTab backlogTab
	sprints           map[sprintTab]*Sprint
	initialLoading    int // Count of initial sprint loads pending
	width             int // Terminal width
	height            int // Terminal height
	// Cursor and scroll are synchronized with current WorkItemList
	cursor       int
	scrollOffset int
	// Edit mode fields
	editTitleInput       textinput.Model
	editDescriptionInput textarea.Model
	editFieldCursor      int // Which field is currently focused (0=title, 1=description)
	editFieldCount       int // Total number of editable fields
	// Create mode fields
	createInput     textinput.Model
	createInsertPos int    // Position in tree to insert
	createAfter     bool   // true='a', false='i'
	createParentID  *int   // nil=parent level, int=child of parent
	createDepth     int    // Tree depth for rendering
	createIsLast    []bool // Tree prefix info for rendering
	createdItemID   int    // Track newly created item for cursor jump
	// Delete mode fields
	deleteItemID    int    // ID of item to delete
	deleteItemTitle string // Title of item to delete (for confirmation message)
	// Batch selection fields
	selectedItems       map[int]bool // Set of selected work item IDs
	batchOperationCount int          // Track pending batch operations
	// UI styles
	styles Styles // Centralized styles for the application
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
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
		state:                listView,
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
