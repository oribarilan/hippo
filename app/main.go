package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

type viewState int

const (
	listView viewState = iota
	detailView
	statePickerView
	findView
	filterView
	helpView
	editView
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
	tasks             []WorkItem
	sprintTasks       []WorkItem                // Tasks for sprint mode only
	backlogTasks      map[backlogTab][]WorkItem // Tasks per backlog tab
	filteredTasks     []WorkItem
	cursor            int
	scrollOffset      int // Scroll offset for list view
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
	filterActive      bool
	statusMessage     string
	lastActionLog     string    // Log line showing the result of the last action
	lastActionTime    time.Time // Timestamp of the last action
	currentMode       appMode
	currentTab        sprintTab
	currentBacklogTab backlogTab
	sprints           map[sprintTab]*Sprint
	sprintCounts      map[sprintTab]int   // Total count per sprint
	sprintLoaded      map[sprintTab]int   // Loaded count per sprint
	sprintAttempted   map[sprintTab]bool  // Whether we've attempted to load this sprint
	backlogCounts     map[backlogTab]int  // Total count per backlog tab
	backlogLoaded     map[backlogTab]int  // Loaded count per backlog tab
	backlogAttempted  map[backlogTab]bool // Whether we've attempted to load this backlog tab
	initialLoading    int                 // Count of initial sprint loads pending
	width             int                 // Terminal width
	height            int                 // Terminal height
	// Edit mode fields
	editTitleInput       textinput.Model
	editDescriptionInput textarea.Model
	editFieldCursor      int // Which field is currently focused (0=title, 1=description)
	editFieldCount       int // Total number of editable fields
}

type WorkItem struct {
	ID            int
	Title         string
	State         string
	AssignedTo    string
	WorkItemType  string
	Description   string
	Tags          string
	Priority      int
	CreatedDate   string
	ChangedDate   string
	IterationPath string
	ParentID      *int
	Children      []*WorkItem
	Comments      string // Discussion/History
}

// TreeItem represents a flattened tree view item with depth information
type TreeItem struct {
	WorkItem *WorkItem
	Depth    int
	IsLast   []bool // Track if ancestor at each level is the last child
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

	return model{
		tasks:                []WorkItem{},
		sprintTasks:          []WorkItem{},
		backlogTasks:         make(map[backlogTab][]WorkItem),
		filteredTasks:        []WorkItem{},
		cursor:               0,
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
		sprintCounts:         make(map[sprintTab]int),
		sprintLoaded:         make(map[sprintTab]int),
		sprintAttempted:      make(map[sprintTab]bool),
		backlogCounts:        make(map[backlogTab]int),
		backlogLoaded:        make(map[backlogTab]int),
		backlogAttempted:     make(map[backlogTab]bool),
		editTitleInput:       editTitleInput,
		editDescriptionInput: editDescriptionInput,
		editFieldCursor:      0,
		editFieldCount:       2, // Title and Description only
	}
}

type tasksLoadedMsg struct {
	tasks         []WorkItem
	err           error
	client        *AzureDevOpsClient
	totalCount    int
	append        bool
	forTab        *sprintTab  // Which sprint tab these tasks are for
	forBacklogTab *backlogTab // Which backlog tab these tasks are for
}

type stateUpdatedMsg struct {
	err error
}

type workItemUpdatedMsg struct {
	err error
}

type workItemRefreshedMsg struct {
	workItem *WorkItem
	err      error
}

type statesLoadedMsg struct {
	states          []string
	stateCategories map[string]string
	err             error
}

type sprintsLoadedMsg struct {
	previousSprint *Sprint
	currentSprint  *Sprint
	nextSprint     *Sprint
	err            error
	client         *AzureDevOpsClient
}

func loadTasks(client *AzureDevOpsClient) tea.Cmd {
	return loadTasksForSprint(client, nil, "", 30, nil)
}

func loadTasksForSprint(client *AzureDevOpsClient, excludeIDs []int, sprintPath string, limit int, forTab *sprintTab) tea.Cmd {
	return func() tea.Msg {
		tasks, err := client.GetWorkItemsExcluding(excludeIDs, sprintPath, limit)
		if err != nil {
			return tasksLoadedMsg{err: err, append: len(excludeIDs) > 0, forTab: forTab}
		}

		totalCount, countErr := client.GetWorkItemsCountForSprint(sprintPath)
		if countErr != nil {
			// If count fails, just use the tasks we got
			totalCount = len(tasks)
		}

		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: len(excludeIDs) > 0, forTab: forTab}
	}
}

func loadInitialTasksForSprint(client *AzureDevOpsClient, sprintPath string, tab sprintTab) tea.Cmd {
	tabCopy := tab
	return loadTasksForSprint(client, nil, sprintPath, 10, &tabCopy)
}

func loadTasksForBacklogTab(client *AzureDevOpsClient, tab backlogTab, currentSprintPath string) tea.Cmd {
	return func() tea.Msg {
		var tasks []WorkItem
		var err error
		var totalCount int

		switch tab {
		case recentBacklog:
			tasks, err = client.GetRecentBacklogItems(30) // Load 30 items initially
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItems(currentSprintPath, 30)
			if err == nil {
				totalCount, _ = client.GetAbandonedWorkItemsCount(currentSprintPath)
			}
		}

		if err != nil {
			return tasksLoadedMsg{err: err, append: false, forTab: nil, forBacklogTab: &tab}
		}

		if totalCount == 0 {
			totalCount = len(tasks)
		}

		tabCopy := tab
		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: false, forTab: nil, forBacklogTab: &tabCopy}
	}
}

func loadMoreBacklogItems(client *AzureDevOpsClient, tab backlogTab, currentSprintPath string, excludeIDs []int) tea.Cmd {
	return func() tea.Msg {
		var tasks []WorkItem
		var err error
		var totalCount int

		switch tab {
		case recentBacklog:
			tasks, err = client.GetRecentBacklogItemsExcluding(excludeIDs, 30)
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItemsExcluding(excludeIDs, currentSprintPath, 30)
			if err == nil {
				totalCount, _ = client.GetAbandonedWorkItemsCount(currentSprintPath)
			}
		}

		if err != nil {
			return tasksLoadedMsg{err: err, append: true, forTab: nil, forBacklogTab: &tab}
		}

		if totalCount == 0 {
			totalCount = len(tasks)
		}

		tabCopy := tab
		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: true, forTab: nil, forBacklogTab: &tabCopy}
	}
}

func loadWorkItemStates(client *AzureDevOpsClient, workItemType string) tea.Cmd {
	return func() tea.Msg {
		states, categories, err := client.GetWorkItemTypeStates(workItemType)
		return statesLoadedMsg{states: states, stateCategories: categories, err: err}
	}
}

func loadSprints(client *AzureDevOpsClient) tea.Cmd {
	return func() tea.Msg {
		prev, curr, next, err := client.GetCurrentAndAdjacentSprints()

		var prevSprint, currSprint, nextSprint *Sprint

		if prev != nil && prev.Name != nil && prev.Path != nil {
			prevSprint = &Sprint{
				Name: *prev.Name,
				Path: *prev.Path,
			}
			if prev.Attributes != nil {
				if prev.Attributes.StartDate != nil {
					prevSprint.StartDate = prev.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if prev.Attributes.FinishDate != nil {
					prevSprint.EndDate = prev.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		if curr != nil && curr.Name != nil && curr.Path != nil {
			currSprint = &Sprint{
				Name: *curr.Name,
				Path: *curr.Path,
			}
			if curr.Attributes != nil {
				if curr.Attributes.StartDate != nil {
					currSprint.StartDate = curr.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if curr.Attributes.FinishDate != nil {
					currSprint.EndDate = curr.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		if next != nil && next.Name != nil && next.Path != nil {
			nextSprint = &Sprint{
				Name: *next.Name,
				Path: *next.Path,
			}
			if next.Attributes != nil {
				if next.Attributes.StartDate != nil {
					nextSprint.StartDate = next.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if next.Attributes.FinishDate != nil {
					nextSprint.EndDate = next.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		return sprintsLoadedMsg{
			previousSprint: prevSprint,
			currentSprint:  currSprint,
			nextSprint:     nextSprint,
			err:            err,
			client:         client,
		}
	}
}

func updateWorkItemState(client *AzureDevOpsClient, workItemID int, newState string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(workItemID, newState)
		return stateUpdatedMsg{err: err}
	}
}

func updateWorkItem(client *AzureDevOpsClient, workItemID int, updates map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItem(workItemID, updates)
		return workItemUpdatedMsg{err: err}
	}
}

func refreshWorkItem(client *AzureDevOpsClient, workItemID int) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.GetWorkItemByID(workItemID)
		return workItemRefreshedMsg{workItem: workItem, err: err}
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
		// Handle help view
		if m.state == helpView {
			switch msg.String() {
			case "esc", "?":
				m.state = listView
				return m, nil
			}
		}

		// Handle filter input (filter existing list by title/ID)
		if m.state == filterView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.filterActive = false
				m.filterInput.SetValue("")
				m.filteredTasks = nil
				m.cursor = 0
				return m, nil
			case "enter":
				// If there are results, open the selected item in detail view
				if len(m.filteredTasks) > 0 && m.cursor < len(m.filteredTasks) {
					// Get the filtered tasks as tree items to respect the cursor position
					visibleTasks := m.getVisibleTasks()
					if len(visibleTasks) > 0 && m.cursor < len(visibleTasks) {
						treeItems := m.getVisibleTreeItems()
						if m.cursor < len(treeItems) {
							m.selectedTask = treeItems[m.cursor].WorkItem
							m.selectedTaskID = m.selectedTask.ID
							m.state = detailView
							m.filterActive = true
						}
					}
				} else {
					// Just close filter and keep filter active
					m.state = listView
					m.filterActive = true
					m.cursor = 0
				}
				return m, nil
			case "up", "ctrl+k", "ctrl+p":
				// Navigate up in filtered results
				if m.cursor > 0 {
					m.cursor--
					m.adjustScrollOffset()
				}
				return m, nil
			case "down", "ctrl+j", "ctrl+n":
				// Navigate down in filtered results
				treeItems := m.getVisibleTreeItems()
				if m.cursor < len(treeItems)-1 {
					m.cursor++
					m.adjustScrollOffset()
				}
				return m, nil
			case "ctrl+u", "pgup":
				// Jump up half page
				m.cursor = max(0, m.cursor-10)
				m.adjustScrollOffset()
				return m, nil
			case "ctrl+d", "pgdown":
				// Jump down half page
				treeItems := m.getVisibleTreeItems()
				m.cursor = min(len(treeItems)-1, m.cursor+10)
				m.adjustScrollOffset()
				return m, nil
			default:
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filterSearch()
				// Reset cursor when filter changes
				m.cursor = 0
				return m, cmd
			}
		}

		// Handle find input (search with dedicated query)
		if m.state == findView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.findInput.SetValue("")
				return m, nil
			case "enter":
				// For now, just go back to list view
				// Could implement custom queries here
				m.state = listView
				m.findInput.SetValue("")
				if m.client != nil {
					m.loading = true
					return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
				}
				return m, nil
			default:
				m.findInput, cmd = m.findInput.Update(msg)
				return m, cmd
			}
		}

		// Handle state picker
		if m.state == statePickerView {
			switch msg.String() {
			case "esc":
				m.state = detailView
				m.stateCursor = 0
				return m, nil
			case "up", "k":
				if m.stateCursor > 0 {
					m.stateCursor--
				}
			case "down", "j":
				if m.stateCursor < len(m.availableStates)-1 {
					m.stateCursor++
				}
			case "ctrl+u", "pgup":
				// Jump up half page
				m.stateCursor = max(0, m.stateCursor-10)
			case "ctrl+d", "pgdown":
				// Jump down half page
				m.stateCursor = min(len(m.availableStates)-1, m.stateCursor+10)
			case "enter":
				if m.selectedTask != nil && m.client != nil {
					newState := m.availableStates[m.stateCursor]
					m.loading = true
					m.statusMessage = fmt.Sprintf("Updating state to %s...", newState)
					return m, tea.Batch(
						updateWorkItemState(m.client, m.selectedTask.ID, newState),
						m.spinner.Tick,
					)
				}
				return m, nil
			}
			return m, nil
		}

		// Handle edit view
		if m.state == editView {
			switch msg.String() {
			case "esc":
				// Cancel edit and return to detail view
				m.state = detailView
				return m, nil
			case "tab":
				// Move to next field
				m.editFieldCursor = (m.editFieldCursor + 1) % m.editFieldCount
				m.focusEditField()
				return m, nil
			case "shift+tab":
				// Move to previous field
				m.editFieldCursor--
				if m.editFieldCursor < 0 {
					m.editFieldCursor = m.editFieldCount - 1
				}
				m.focusEditField()
				return m, nil
			case "ctrl+s":
				// Save changes
				if m.selectedTask != nil && m.client != nil {
					updates := make(map[string]interface{})

					// Collect values from inputs
					if title := m.editTitleInput.Value(); title != "" && title != m.selectedTask.Title {
						updates["title"] = title
					}
					if desc := m.editDescriptionInput.Value(); desc != m.selectedTask.Description {
						updates["description"] = desc
					}

					// Only update if there are changes
					if len(updates) > 0 {
						m.loading = true
						m.statusMessage = "Saving changes..."
						return m, tea.Batch(
							updateWorkItem(m.client, m.selectedTask.ID, updates),
							m.spinner.Tick,
						)
					} else {
						// No changes, just return to detail view
						m.state = detailView
						return m, nil
					}
				}
				return m, nil
			default:
				// Update the focused input field
				var cmd tea.Cmd
				switch m.editFieldCursor {
				case 0:
					m.editTitleInput, cmd = m.editTitleInput.Update(msg)
				case 1:
					m.editDescriptionInput, cmd = m.editDescriptionInput.Update(msg)
				}
				return m, cmd
			}
		}

		// Global hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "1":
			// Switch to Sprint Mode
			if m.state == listView && m.currentMode != sprintMode {
				m.currentMode = sprintMode
				m.cursor = 0
				m.scrollOffset = 0
				m.setActionLog("Switched to Sprint Mode")
			}
			return m, nil

		case "2":
			// Switch to Backlog Mode
			if m.state == listView && m.currentMode != backlogMode {
				m.currentMode = backlogMode
				m.cursor = 0
				m.scrollOffset = 0
				// Load backlog data if not attempted yet
				if !m.backlogAttempted[m.currentBacklogTab] && m.client != nil {
					m.loading = true
					tab := m.currentBacklogTab
					return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
				}
				m.setActionLog("Switched to Backlog Mode")
			}
			return m, nil

		case "?":
			// Show help modal
			if m.state == helpView {
				m.state = listView
			} else {
				m.state = helpView
			}
			return m, nil

		case "r":
			// Refresh
			if m.client != nil {
				// If we're in detail view, just refresh the selected item
				if m.state == detailView && m.selectedTask != nil {
					m.loading = true
					m.statusMessage = "Refreshing item..."
					m.setActionLog(fmt.Sprintf("Refreshing #%d...", m.selectedTask.ID))
					return m, tea.Batch(refreshWorkItem(m.client, m.selectedTask.ID), m.spinner.Tick)
				}
				// Otherwise, refresh everything based on current mode
				m.loading = true
				m.statusMessage = "Refreshing..."
				m.setActionLog("Refreshing data...")
				m.filterActive = false
				m.filterInput.SetValue("")
				m.filteredTasks = nil

				if m.currentMode == sprintMode {
					// Clear sprint data
					m.sprintTasks = nil
					m.sprintCounts = make(map[sprintTab]int)
					m.sprintLoaded = make(map[sprintTab]int)
					m.sprintAttempted = make(map[sprintTab]bool)
					return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
				} else {
					// Clear backlog data
					m.backlogTasks = make(map[backlogTab][]WorkItem)
					m.backlogCounts = make(map[backlogTab]int)
					m.backlogLoaded = make(map[backlogTab]int)
					m.backlogAttempted = make(map[backlogTab]bool)
					// Reload current backlog tab
					tab := m.currentBacklogTab
					return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
				}
			}
			return m, nil

		case "o":
			// Open in browser
			var workItemID int
			if m.state == detailView && m.selectedTask != nil {
				workItemID = m.selectedTask.ID
			} else if m.state == listView {
				treeItems := m.getVisibleTreeItems()
				if len(treeItems) > 0 && m.cursor < len(treeItems) {
					workItemID = treeItems[m.cursor].WorkItem.ID
				}
			}
			if workItemID > 0 {
				openInBrowser(m.organizationURL, m.projectName, workItemID)
				m.setActionLog(fmt.Sprintf("Opened #%d in browser", workItemID))
			}
			return m, nil

		case "s":
			// Change state - only in detail view
			if m.state == detailView && m.selectedTask != nil && m.client != nil {
				m.loading = true
				m.statusMessage = "Loading states..."
				return m, tea.Batch(
					loadWorkItemStates(m.client, m.selectedTask.WorkItemType),
					m.spinner.Tick,
				)
			}
			return m, nil

		case "e":
			// Enter edit mode - only in detail view
			if m.state == detailView && m.selectedTask != nil {
				// Populate edit fields with current values
				m.editTitleInput.SetValue(m.selectedTask.Title)
				m.editDescriptionInput.SetValue(m.selectedTask.Description)
				m.editFieldCursor = 0
				m.editTitleInput.Focus()
				m.editDescriptionInput.Blur()
				m.state = editView
			}
			return m, nil

		case "/":
			// Filter within existing results
			if m.state == listView {
				m.state = filterView
				m.filterInput.Focus()
				m.filterActive = true // Activate filter immediately
				m.cursor = 0
				m.scrollOffset = 0
			}
			return m, nil

		case "f":
			// Find with dedicated query
			if m.state == listView {
				m.state = findView
				m.findInput.Focus()
			}
			return m, nil
		}

		// List view navigation
		if m.state == listView {
			switch msg.String() {
			case "right", "l":
				// Drill down to detail view
				treeItems := m.getVisibleTreeItems()
				if len(treeItems) > 0 && m.cursor < len(treeItems) {
					m.selectedTask = treeItems[m.cursor].WorkItem
					m.selectedTaskID = m.selectedTask.ID
					m.state = detailView
				}
			case "tab":
				// Cycle through tabs based on current mode
				if m.currentMode == sprintMode {
					m.currentTab = (m.currentTab + 1) % 3
					m.cursor = 0
					m.scrollOffset = 0
					// Load sprint data if not attempted yet
					if !m.sprintAttempted[m.currentTab] && m.sprints[m.currentTab] != nil && m.client != nil {
						sprint := m.sprints[m.currentTab]
						m.loading = true
						tab := m.currentTab
						return m, tea.Batch(loadTasksForSprint(m.client, nil, sprint.Path, 10, &tab), m.spinner.Tick)
					}
				} else if m.currentMode == backlogMode {
					m.currentBacklogTab = (m.currentBacklogTab + 1) % 2
					m.cursor = 0
					m.scrollOffset = 0
					// Load backlog data if not attempted yet
					if !m.backlogAttempted[m.currentBacklogTab] && m.client != nil {
						m.loading = true
						tab := m.currentBacklogTab
						return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
					}
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					m.adjustScrollOffset()
				}
			case "down", "j":
				treeItems := m.getVisibleTreeItems()
				maxCursor := len(treeItems) - 1
				// If there are more items to load, allow cursor to go to the "Load More" item
				if m.hasMoreItems() {
					maxCursor = len(treeItems)
				}
				if m.cursor < maxCursor {
					m.cursor++
					m.adjustScrollOffset()
				}
			case "ctrl+u", "pgup":
				// Jump up half page
				m.cursor = max(0, m.cursor-10)
				m.adjustScrollOffset()
			case "ctrl+d", "pgdown":
				// Jump down half page
				treeItems := m.getVisibleTreeItems()
				maxCursor := len(treeItems) - 1
				if m.hasMoreItems() {
					maxCursor = len(treeItems)
				}
				m.cursor = min(maxCursor, m.cursor+10)
				m.adjustScrollOffset()
			case "enter":
				treeItems := m.getVisibleTreeItems()
				// Check if cursor is on "Load More" item
				if m.cursor == len(treeItems) && m.hasMoreItems() {
					if m.client != nil {
						m.loadingMore = true

						if m.currentMode == sprintMode {
							// Load more items by excluding already loaded IDs for current sprint
							// Get current sprint path
							sprint := m.sprints[m.currentTab]
							sprintPath := ""
							if sprint != nil {
								sprintPath = sprint.Path
							}

							// Collect all currently loaded IDs in this sprint to exclude
							excludeIDs := make([]int, 0)
							for _, task := range m.sprintTasks {
								if task.IterationPath == sprintPath {
									excludeIDs = append(excludeIDs, task.ID)
								}
							}

							tab := m.currentTab
							return m, tea.Batch(loadTasksForSprint(m.client, excludeIDs, sprintPath, 30, &tab), m.spinner.Tick)
						} else if m.currentMode == backlogMode {
							// Load more items for backlog mode
							// Collect all currently loaded IDs in this backlog tab to exclude
							excludeIDs := make([]int, 0)
							for _, task := range m.backlogTasks[m.currentBacklogTab] {
								excludeIDs = append(excludeIDs, task.ID)
							}
							return m, tea.Batch(loadMoreBacklogItems(m.client, m.currentBacklogTab, m.getCurrentSprintPath(), excludeIDs), m.spinner.Tick)
						}
					}
				} else if len(treeItems) > 0 && m.cursor < len(treeItems) {
					m.selectedTask = treeItems[m.cursor].WorkItem
					m.selectedTaskID = m.selectedTask.ID
					m.state = detailView
				}
			}
		}

		// Detail view navigation
		if m.state == detailView {
			switch msg.String() {
			case "esc", "backspace", "left", "h":
				m.state = listView
			}
		}

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.statusMessage = ""
			m.loadingMore = false
			// If this was part of initial loading, decrement counter
			if m.initialLoading > 0 {
				m.initialLoading--
				if m.initialLoading == 0 {
					m.loading = false
					// Set log message after all initial sprints are loaded
					m.setActionLog("Loaded previous, current, and next sprint")
				}
			} else {
				m.loading = false
			}
		} else {
			// Handle sprint tab loading
			if msg.forTab != nil {
				targetTab := *msg.forTab

				if msg.append {
					// Append new tasks to existing ones (load more scenario)
					m.sprintTasks = append(m.sprintTasks, msg.tasks...)

					// Update sprint-specific counts
					m.sprintLoaded[targetTab] = m.sprintLoaded[targetTab] + len(msg.tasks)
					m.sprintCounts[targetTab] = msg.totalCount

					m.setActionLog(fmt.Sprintf("Loaded %d more items", len(msg.tasks)))
					m.loading = false
					m.loadingMore = false
				} else {
					// Initial load or replace
					if len(m.sprintTasks) == 0 {
						// Very first load
						m.sprintTasks = msg.tasks
					} else {
						// Append to existing (for other sprint loads)
						m.sprintTasks = append(m.sprintTasks, msg.tasks...)
					}
					m.filteredTasks = nil
					if targetTab == m.currentTab && m.currentMode == sprintMode {
						m.cursor = 0
						m.scrollOffset = 0
					}
					m.statusMessage = ""

					// All tasks in msg.tasks are for this sprint (filtered by API)
					m.sprintLoaded[targetTab] = len(msg.tasks)
					m.sprintCounts[targetTab] = msg.totalCount
					m.sprintAttempted[targetTab] = true // Mark as attempted

					// If this was part of initial loading, decrement counter
					if m.initialLoading > 0 {
						m.initialLoading--
						if m.initialLoading == 0 {
							m.loading = false
							// Set log message after all initial sprints are loaded
							m.setActionLog("Loaded previous, current, and next sprint")
						}
					} else {
						m.loading = false
						// Only set individual load message if not part of initial loading
						m.setActionLog(fmt.Sprintf("Loaded %d items", len(msg.tasks)))
					}
				}
			} else if msg.forBacklogTab != nil {
				// Handle backlog tab loading
				targetTab := *msg.forBacklogTab

				if msg.append {
					// Append new tasks to existing ones (load more scenario)
					existing := m.backlogTasks[targetTab]
					m.backlogTasks[targetTab] = append(existing, msg.tasks...)

					// Update backlog-specific counts
					m.backlogLoaded[targetTab] = m.backlogLoaded[targetTab] + len(msg.tasks)
					m.backlogCounts[targetTab] = msg.totalCount

					m.setActionLog(fmt.Sprintf("Loaded %d more items", len(msg.tasks)))
					m.loading = false
					m.loadingMore = false
				} else {
					// Store tasks for specific backlog tab (each tab is independent)
					m.backlogTasks[targetTab] = msg.tasks

					m.filteredTasks = nil
					if targetTab == m.currentBacklogTab && m.currentMode == backlogMode {
						m.cursor = 0
						m.scrollOffset = 0
					}
					m.statusMessage = ""

					// Update backlog-specific counts
					m.backlogLoaded[targetTab] = len(msg.tasks)
					m.backlogCounts[targetTab] = msg.totalCount
					m.backlogAttempted[targetTab] = true // Mark as attempted

					m.loading = false
					m.setActionLog(fmt.Sprintf("Loaded %d items", len(msg.tasks)))
				}
			}

			// Store the client if it was passed
			if msg.client != nil {
				m.client = msg.client
			}
		}

	case stateUpdatedMsg:
		m.loading = false
		m.statusMessage = ""
		oldState := ""
		taskTitle := ""
		if m.selectedTask != nil {
			oldState = m.selectedTask.State
			taskTitle = m.selectedTask.Title
		}
		newState := ""
		if m.stateCursor < len(m.availableStates) {
			newState = m.availableStates[m.stateCursor]
		}

		m.state = listView
		m.stateCursor = 0
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error updating state: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error updating state: %v", msg.err))
		} else {
			m.statusMessage = "State updated successfully!"
			if taskTitle != "" && oldState != "" && newState != "" {
				m.setActionLog(fmt.Sprintf("Updated \"%s\": %s → %s", taskTitle, oldState, newState))
			} else {
				m.setActionLog("State updated successfully")
			}
			// Refresh the list
			if m.client != nil {
				m.loading = true
				return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
			}
		}

	case workItemUpdatedMsg:
		m.loading = false
		m.statusMessage = ""
		taskTitle := ""
		if m.selectedTask != nil {
			taskTitle = m.selectedTask.Title
		}

		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error updating work item: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error updating work item: %v", msg.err))
			m.state = editView // Stay in edit view on error
		} else {
			m.statusMessage = "Work item updated successfully!"
			if taskTitle != "" {
				m.setActionLog(fmt.Sprintf("Updated \"%s\"", taskTitle))
			} else {
				m.setActionLog("Work item updated successfully")
			}
			m.state = detailView // Return to detail view on success
			// Refresh the list
			if m.client != nil {
				m.loading = true
				return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
			}
		}

	case statesLoadedMsg:
		m.loading = false
		m.statusMessage = ""
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading states: %v", msg.err)
		} else {
			m.availableStates = msg.states
			m.stateCategories = msg.stateCategories
			m.state = statePickerView
			m.stateCursor = 0
		}

	case workItemRefreshedMsg:
		m.loading = false
		m.statusMessage = ""
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error refreshing item: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error refreshing item: %v", msg.err))
		} else if msg.workItem != nil {
			// Update the selected task in detail view
			m.selectedTask = msg.workItem

			// Also update the item in the appropriate tasks list
			tasks := m.getCurrentTasks()
			for i := range tasks {
				if tasks[i].ID == msg.workItem.ID {
					// Preserve children since they're not loaded in single item refresh
					existingChildren := tasks[i].Children
					tasks[i] = *msg.workItem
					tasks[i].Children = existingChildren
					break
				}
			}
			// Update the appropriate list
			m.setCurrentTasks(tasks)

			m.setActionLog(fmt.Sprintf("Refreshed #%d", msg.workItem.ID))
		}

	case sprintsLoadedMsg:
		// Store client first
		if msg.client != nil {
			m.client = msg.client
		}

		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading sprints: %v", msg.err)
		} else {
			needsReload := len(m.sprints) == 0 // First time loading sprints

			if msg.previousSprint != nil {
				m.sprints[previousSprint] = msg.previousSprint
			}
			if msg.currentSprint != nil {
				m.sprints[currentSprint] = msg.currentSprint
			}
			if msg.nextSprint != nil {
				m.sprints[nextSprint] = msg.nextSprint
			}

			// If this is the first time we loaded sprints, load initial data for each sprint
			if needsReload && m.client != nil {
				var loadCmds []tea.Cmd

				// Load 10 items for each sprint
				sprintCount := 0
				for tab, sprint := range m.sprints {
					if sprint != nil {
						loadCmds = append(loadCmds, loadInitialTasksForSprint(m.client, sprint.Path, tab))
						sprintCount++
					}
				}

				if len(loadCmds) > 0 {
					m.loading = true
					m.initialLoading = sprintCount // Track how many sprints we're loading
					loadCmds = append(loadCmds, m.spinner.Tick)
					return m, tea.Batch(loadCmds...)
				}
			}
		}

	case spinner.TickMsg:
		if m.loading || m.loadingMore {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// buildTreeStructure organizes work items into a parent-child hierarchy
func buildTreeStructure(items []WorkItem) []*WorkItem {
	// Create a map of all items by ID for quick lookup
	itemMap := make(map[int]*WorkItem)
	for i := range items {
		itemMap[items[i].ID] = &items[i]
		items[i].Children = nil // Reset children
	}

	// Build parent-child relationships and collect root items
	var roots []*WorkItem
	for i := range items {
		item := &items[i]
		if item.ParentID != nil {
			// This item has a parent
			if parent, exists := itemMap[*item.ParentID]; exists {
				// Parent exists in our list, add as child
				parent.Children = append(parent.Children, item)
			} else {
				// Parent not in our list, treat as root
				roots = append(roots, item)
			}
		} else {
			// No parent, this is a root item
			roots = append(roots, item)
		}
	}

	return roots
}

// flattenTree converts a tree structure into a flat list with depth information
func flattenTree(roots []*WorkItem) []TreeItem {
	var result []TreeItem

	var traverse func(item *WorkItem, depth int, isLast []bool)
	traverse = func(item *WorkItem, depth int, isLast []bool) {
		result = append(result, TreeItem{
			WorkItem: item,
			Depth:    depth,
			IsLast:   append([]bool{}, isLast...), // Copy the slice
		})

		// Recursively add children
		for i, child := range item.Children {
			// Create new isLast slice for this child
			childIsLast := append([]bool{}, isLast...)
			childIsLast = append(childIsLast, i == len(item.Children)-1)
			traverse(child, depth+1, childIsLast)
		}
	}

	for i, root := range roots {
		isLast := []bool{i == len(roots)-1}
		traverse(root, 0, isLast)
	}

	return result
}

// getTreePrefix returns the tree drawing prefix for a tree item with enhanced styling
func getTreePrefix(treeItem TreeItem) string {
	if treeItem.Depth == 0 {
		return ""
	}

	var prefix strings.Builder

	// Draw vertical lines and spaces for each level except the last
	for i := 0; i < treeItem.Depth-1; i++ {
		if treeItem.IsLast[i] {
			prefix.WriteString("    ") // No vertical line if parent was last
		} else {
			prefix.WriteString("│   ") // Vertical line if parent has more siblings
		}
	}

	// Draw the connector for this item with rounded corners
	if len(treeItem.IsLast) > 0 && treeItem.IsLast[len(treeItem.IsLast)-1] {
		prefix.WriteString("╰── ") // Last child with rounded corner
	} else {
		prefix.WriteString("├── ") // Not last child
	}

	return prefix.String()
}

// getWorkItemIcon returns an icon based on the work item type
func getWorkItemIcon(workItemType string) string {
	switch strings.ToLower(workItemType) {
	case "task":
		return "✓"
	default:
		return "•"
	}
}

func (m *model) filterSearch() {
	query := strings.ToLower(m.filterInput.Value())
	if query == "" {
		m.filteredTasks = nil
		return
	}

	currentTasks := m.getCurrentTasks()
	var filtered []WorkItem
	for _, task := range currentTasks {
		if strings.Contains(strings.ToLower(task.Title), query) ||
			strings.Contains(fmt.Sprintf("%d", task.ID), query) {
			filtered = append(filtered, task)
		}
	}
	m.filteredTasks = filtered
}

func (m model) getVisibleTasks() []WorkItem {
	tasks := m.getCurrentTasks()

	// Apply filter if active
	if m.filterActive && m.filteredTasks != nil {
		tasks = m.filteredTasks
	}

	// Filter based on current mode
	if m.currentMode == sprintMode {
		// Apply sprint filter
		sprint := m.sprints[m.currentTab]
		if sprint == nil || sprint.Path == "" {
			// If no sprint data available, show all tasks
			return tasks
		}

		var filtered []WorkItem
		for _, task := range tasks {
			if task.IterationPath == sprint.Path {
				filtered = append(filtered, task)
			}
		}

		return filtered
	} else if m.currentMode == backlogMode {
		// In backlog mode, tasks are already filtered by the query
		// No additional filtering needed
		return tasks
	}

	return tasks
}

// getVisibleTreeItems returns visible tasks organized as a tree structure
func (m model) getVisibleTreeItems() []TreeItem {
	visibleTasks := m.getVisibleTasks()
	if len(visibleTasks) == 0 {
		return []TreeItem{}
	}

	// Build tree structure
	roots := buildTreeStructure(visibleTasks)

	// Flatten tree for display
	return flattenTree(roots)
}

func (m model) hasMoreItems() bool {
	if m.currentMode == sprintMode {
		// Check if there are more items in the current sprint that we haven't loaded yet
		loaded := m.sprintLoaded[m.currentTab]
		total := m.sprintCounts[m.currentTab]
		return loaded < total
	} else if m.currentMode == backlogMode {
		// Check if there are more items in the current backlog tab
		loaded := m.backlogLoaded[m.currentBacklogTab]
		total := m.backlogCounts[m.currentBacklogTab]
		return loaded < total
	}
	return false
}

func (m model) getRemainingCount() int {
	if m.currentMode == sprintMode {
		loaded := m.sprintLoaded[m.currentTab]
		total := m.sprintCounts[m.currentTab]
		return total - loaded
	} else if m.currentMode == backlogMode {
		loaded := m.backlogLoaded[m.currentBacklogTab]
		total := m.backlogCounts[m.currentBacklogTab]
		return total - loaded
	}
	return 0
}

// getParentTask finds and returns the parent task if it exists
func (m model) getParentTask(task *WorkItem) *WorkItem {
	if task == nil || task.ParentID == nil {
		return nil
	}

	// Search through current tasks list to find the parent
	currentTasks := m.getCurrentTasks()
	for i := range currentTasks {
		if currentTasks[i].ID == *task.ParentID {
			return &currentTasks[i]
		}
	}

	return nil
}

// formatDateTime formats a datetime string into a human-readable format
func formatDateTime(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Parse the date string (format: 2006-01-02T15:04:05)
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// Try without the time part
		t, err = time.Parse("2006-01-02", dateStr[:10])
		if err != nil {
			return dateStr
		}
	}

	// Format as "Jan 2, 2006 3:04 PM"
	return t.Format("Jan 2, 2006 3:04 PM")
}

// getRelativeTime returns a human-readable relative time string
func getRelativeTime(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Parse the date string (format: 2006-01-02T15:04:05)
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// Try without the time part
		t, err = time.Parse("2006-01-02", dateStr[:10])
		if err != nil {
			return ""
		}
	}

	duration := time.Since(t)
	days := int(duration.Hours() / 24)
	weeks := days / 7

	if days < 1 {
		return "(< day ago)"
	} else if days == 1 {
		return "(1 day ago)"
	} else if weeks == 0 {
		return fmt.Sprintf("(%d days ago)", days)
	} else if weeks == 1 {
		return "(1 week ago)"
	} else if weeks <= 10 {
		return fmt.Sprintf("(%d weeks ago)", weeks)
	} else {
		return "(10+ weeks ago)"
	}
}

// buildDetailContent creates the content string for the detail view
func (m model) buildDetailContent() string {
	if m.selectedTask == nil {
		return ""
	}

	task := m.selectedTask

	// Styles
	cardStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(m.width - 4)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(15)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginTop(1).
		MarginBottom(1)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")).
		Italic(true).
		MarginTop(1).
		MarginBottom(1)

	// Build the card content
	var cardContent strings.Builder

	// Header with ID and Title
	cardContent.WriteString(headerStyle.Render(fmt.Sprintf("#%d - %s", task.ID, task.Title)))
	cardContent.WriteString("\n\n")

	// Parent info if exists
	if parent := m.getParentTask(task); parent != nil {
		cardContent.WriteString(labelStyle.Render("Parent:"))
		cardContent.WriteString(valueStyle.Render(fmt.Sprintf("#%d - %s", parent.ID, parent.Title)))
		cardContent.WriteString("\n")
	}

	// Basic Info
	cardContent.WriteString(labelStyle.Render("Type:"))
	cardContent.WriteString(valueStyle.Render(task.WorkItemType))
	cardContent.WriteString("\n")

	cardContent.WriteString(labelStyle.Render("State:"))
	cardContent.WriteString(valueStyle.Render(task.State))
	cardContent.WriteString("\n")

	if task.AssignedTo != "" {
		cardContent.WriteString(labelStyle.Render("Assigned To:"))
		cardContent.WriteString(valueStyle.Render(task.AssignedTo))
		cardContent.WriteString("\n")
	}

	if task.Priority > 0 {
		cardContent.WriteString(labelStyle.Render("Priority:"))
		cardContent.WriteString(valueStyle.Render(fmt.Sprintf("%d", task.Priority)))
		cardContent.WriteString("\n")
	}

	if task.Tags != "" {
		cardContent.WriteString(labelStyle.Render("Tags:"))
		cardContent.WriteString(valueStyle.Render(task.Tags))
		cardContent.WriteString("\n")
	}

	if task.IterationPath != "" {
		cardContent.WriteString(labelStyle.Render("Sprint:"))
		// Extract just the sprint name from the path
		parts := strings.Split(task.IterationPath, "\\")
		sprintName := parts[len(parts)-1]
		cardContent.WriteString(valueStyle.Render(sprintName))
		cardContent.WriteString("\n")
	}

	if task.CreatedDate != "" {
		formattedDate := formatDateTime(task.CreatedDate)
		relativeTime := getRelativeTime(task.CreatedDate)
		dateValue := formattedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", formattedDate, relativeTime)
		}
		cardContent.WriteString(labelStyle.Render("Created:"))
		cardContent.WriteString(valueStyle.Render(dateValue))
		cardContent.WriteString("\n")
	}

	if task.ChangedDate != "" {
		formattedDate := formatDateTime(task.ChangedDate)
		relativeTime := getRelativeTime(task.ChangedDate)
		dateValue := formattedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", formattedDate, relativeTime)
		}
		cardContent.WriteString(labelStyle.Render("Last Updated:"))
		cardContent.WriteString(valueStyle.Render(dateValue))
		cardContent.WriteString("\n")
	}

	// Description Section
	if task.Description != "" {
		cardContent.WriteString("\n")
		cardContent.WriteString(sectionStyle.Render("Description"))
		cardContent.WriteString("\n")
		cardContent.WriteString(descriptionStyle.Render(task.Description))
	}

	// Comments / Discussion Section
	if task.Comments != "" {
		cardContent.WriteString("\n")
		cardContent.WriteString(sectionStyle.Render("Comments / Discussion"))
		cardContent.WriteString("\n")
		cardContent.WriteString(descriptionStyle.Render(task.Comments))
	}

	return cardStyle.Render(cardContent.String())
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

// getStateCategory returns the category for a given state name
func (m model) getStateCategory(state string) string {
	if category, ok := m.stateCategories[state]; ok {
		return category
	}
	// Fallback: try to guess based on common patterns
	stateLower := strings.ToLower(state)
	if strings.Contains(stateLower, "closed") || strings.Contains(stateLower, "done") || strings.Contains(stateLower, "completed") {
		return "Completed"
	}
	if strings.Contains(stateLower, "new") || strings.Contains(stateLower, "proposed") {
		return "Proposed"
	}
	return "InProgress"
}

// getCurrentSprintPath returns the current sprint path if available
func (m model) getCurrentSprintPath() string {
	if sprint := m.sprints[currentSprint]; sprint != nil {
		return sprint.Path
	}
	return ""
}

// getTabHint returns a descriptive hint for the current tab
func (m model) getTabHint() string {
	if m.currentMode == sprintMode {
		sprint := m.sprints[m.currentTab]
		if sprint != nil && sprint.StartDate != "" && sprint.EndDate != "" {
			return fmt.Sprintf("Sprint: %s to %s", sprint.StartDate, sprint.EndDate)
		}
		return ""
	} else {
		// Backlog mode hints
		switch m.currentBacklogTab {
		case recentBacklog:
			return "Items not in any sprint, created or updated in the last 30 days"
		case abandonedWork:
			return "Items not updated in the last 14 days (excluding current sprint)"
		default:
			return ""
		}
	}
}

// getCurrentTasks returns the task list for the current mode
func (m model) getCurrentTasks() []WorkItem {
	if m.currentMode == sprintMode {
		return m.sprintTasks
	}
	// Return tasks for current backlog tab
	if tasks, ok := m.backlogTasks[m.currentBacklogTab]; ok {
		return tasks
	}
	return []WorkItem{}
}

// setCurrentTasks sets the task list for the current mode
func (m *model) setCurrentTasks(tasks []WorkItem) {
	if m.currentMode == sprintMode {
		m.sprintTasks = tasks
	} else {
		m.backlogTasks[m.currentBacklogTab] = tasks
	}
}

// appendCurrentTasks appends to the task list for the current mode
func (m *model) appendCurrentTasks(tasks []WorkItem) {
	if m.currentMode == sprintMode {
		m.sprintTasks = append(m.sprintTasks, tasks...)
	} else {
		existing := m.backlogTasks[m.currentBacklogTab]
		m.backlogTasks[m.currentBacklogTab] = append(existing, tasks...)
	}
}

func (m model) getVisibleTasksCount() int {
	// Get count of visible tasks based on current mode
	if m.currentMode == sprintMode {
		sprint := m.sprints[m.currentTab]
		if sprint == nil || sprint.Path == "" {
			return len(m.sprintTasks)
		}

		count := 0
		for _, task := range m.sprintTasks {
			if task.IterationPath == sprint.Path {
				count++
			}
		}
		return count
	} else {
		// Return count for current backlog tab
		if tasks, ok := m.backlogTasks[m.currentBacklogTab]; ok {
			return len(tasks)
		}
		return 0
	}
}

// getContentHeight returns the available height for rendering work items
// Takes into account title bar, mode/tab selectors, hint, footer, etc.
func (m model) getContentHeight() int {
	// Calculate fixed UI elements height:
	// - Title bar: 3 lines (title + 2 blank lines)
	// - Mode selector: 2 lines (modes + blank line)
	// - Tab selector: 2 lines (tabs + blank line)
	// - Hint (if present): 2 lines (hint + blank line)
	// - Footer: 4 lines (blank + action log + separator + keybindings)

	fixedHeight := 3 + 2 + 2 + 4 // = 11 lines minimum

	// Add hint lines if present
	if m.getTabHint() != "" {
		fixedHeight += 2
	}

	// Add status message lines if present
	if m.statusMessage != "" {
		fixedHeight += 2
	}

	contentHeight := m.height - fixedHeight
	if contentHeight < 5 {
		contentHeight = 5 // Minimum content height
	}

	return contentHeight
}

// adjustScrollOffset adjusts the scroll offset to keep the cursor visible
func (m *model) adjustScrollOffset() {
	contentHeight := m.getContentHeight()

	// Ensure cursor is within visible area
	if m.cursor < m.scrollOffset {
		// Cursor moved above visible area, scroll up
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+contentHeight {
		// Cursor moved below visible area, scroll down
		m.scrollOffset = m.cursor - contentHeight + 1
	}

	// Ensure scroll offset is never negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// renderLogLine renders the action log line if there is one
func (m model) renderLogLine() string {
	if m.lastActionLog == "" {
		return ""
	}

	logStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	timestamp := m.lastActionTime.Format("15:04:05")
	return logStyle.Render(fmt.Sprintf("[%s] %s", timestamp, m.lastActionLog))
}

// setActionLog sets the action log message with the current timestamp
func (m *model) setActionLog(message string) {
	m.lastActionLog = message
	m.lastActionTime = time.Now()
}

// focusEditField focuses the appropriate edit field based on editFieldCursor
func (m *model) focusEditField() {
	// Blur all fields first
	m.editTitleInput.Blur()
	m.editDescriptionInput.Blur()

	// Focus the selected field
	switch m.editFieldCursor {
	case 0:
		m.editTitleInput.Focus()
	case 1:
		m.editDescriptionInput.Focus()
	}
}

// renderTitleBar renders the title bar with the given title text
func (m model) renderTitleBar(title string) string {
	titleBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(m.width).
		Padding(0, 1)

	// Calculate padding to align version to the right
	versionText := version
	availableWidth := m.width - len(title) - len(versionText) - 4 // 4 for padding (2 on each side)
	if availableWidth < 0 {
		availableWidth = 0
	}
	padding := strings.Repeat(" ", availableWidth)

	titleWithVersion := title + padding + versionText
	return titleBarStyle.Render(titleWithVersion) + "\n\n"
}

// renderFooter renders the bottom section with action log and keybindings
func (m model) renderFooter(keybindings string) string {
	var footer strings.Builder

	// Action log line
	footer.WriteString("\n")
	if m.lastActionLog != "" {
		footer.WriteString(m.renderLogLine() + "\n")
	}

	// Separator line
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(m.width)
	separator := separatorStyle.Render(strings.Repeat("─", m.width))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	footer.WriteString(separator + "\n")
	footer.WriteString(helpStyle.Render(keybindings))

	return footer.String()
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n", m.err)
	}

	switch m.state {
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
	case filterView:
		return m.renderFilterView()
	case findView:
		return m.renderFindView()
	case helpView:
		return m.renderHelpView()
	case editView:
		return m.renderEditView()
	default:
		return m.renderListView()
	}
}

func (m model) renderListView() string {
	var content strings.Builder

	// Title bar
	title := "Azure DevOps - Work Items"
	if m.filterActive {
		title += fmt.Sprintf(" (filtered: %d results)", len(m.filteredTasks))
	}
	content.WriteString(m.renderTitleBar(title))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	// State styles based on category
	proposedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")) // Normal gray

	inProgressStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")). // Green
		Bold(true)

	completedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")). // Dimmed gray
		Italic(true)

	removedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")). // Very dim
		Italic(true)

	// Mode and tab styles
	activeModeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2).
		MarginRight(1)

	inactiveModeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2).
		MarginRight(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	// Render mode selector
	modes := []string{}
	if m.currentMode == sprintMode {
		modes = append(modes, activeModeStyle.Render("[1] Sprint"))
		modes = append(modes, inactiveModeStyle.Render("[2] Backlog"))
	} else {
		modes = append(modes, inactiveModeStyle.Render("[1] Sprint"))
		modes = append(modes, activeModeStyle.Render("[2] Backlog"))
	}
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, modes...) + "\n\n")

	// Render tabs based on current mode
	tabs := []string{}

	if m.currentMode == sprintMode {
		prevLabel := "Previous Sprint"
		if sprint := m.sprints[previousSprint]; sprint != nil {
			prevLabel = sprint.Name
		}
		if m.currentTab == previousSprint {
			tabs = append(tabs, activeTabStyle.Render(prevLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(prevLabel))
		}

		currLabel := "Current Sprint"
		if sprint := m.sprints[currentSprint]; sprint != nil {
			currLabel = sprint.Name
		}
		if m.currentTab == currentSprint {
			tabs = append(tabs, activeTabStyle.Render(currLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(currLabel))
		}

		nextLabel := "Next Sprint"
		if sprint := m.sprints[nextSprint]; sprint != nil {
			nextLabel = sprint.Name
		}
		if m.currentTab == nextSprint {
			tabs = append(tabs, activeTabStyle.Render(nextLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(nextLabel))
		}
	} else if m.currentMode == backlogMode {
		// Backlog mode tabs
		if m.currentBacklogTab == recentBacklog {
			tabs = append(tabs, activeTabStyle.Render("Recent Backlog"))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render("Recent Backlog"))
		}

		if m.currentBacklogTab == abandonedWork {
			tabs = append(tabs, activeTabStyle.Render("Abandoned Work"))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render("Abandoned Work"))
		}
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// Show tab hint
	if hint := m.getTabHint(); hint != "" {
		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		content.WriteString(hintStyle.Render(hint) + "\n\n")
	}

	if m.statusMessage != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		content.WriteString(msgStyle.Render(m.statusMessage) + "\n\n")
	}

	// Show loader if loading (but not loadingMore - that shows inline)
	if m.loading && !m.loadingMore {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate\nenter: details • o: open in browser • /: search • f: filter • r: refresh • q: quit"
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()
	if len(treeItems) == 0 {
		content.WriteString("  No tasks found.\n")
	}

	// Style for tree edges with more vibrant color
	edgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")) // Brighter blue-purple
	edgeStyleSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62"))

	// Icon style
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	iconStyleSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62"))

	// Calculate visible range based on scroll offset
	contentHeight := m.getContentHeight()
	startIdx := m.scrollOffset
	endIdx := m.scrollOffset + contentHeight

	// Total items including potential "Load More" item
	totalItems := len(treeItems)
	if m.hasMoreItems() {
		totalItems++
	}

	// Clamp end index
	if endIdx > totalItems {
		endIdx = totalItems
	}

	// Render only visible items
	for i := startIdx; i < endIdx; i++ {
		// Check if this is the "Load More" item
		if i >= len(treeItems) {
			// This is the "Load More" item
			if m.hasMoreItems() {
				remaining := m.getRemainingCount()
				loadMoreStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Italic(true)

				cursor := " "
				loadMoreIdx := len(treeItems)
				if m.cursor == loadMoreIdx {
					cursor = ">"
				}

				var loadMoreText string

				// Show spinner inline if loading more
				if m.loadingMore {
					loadMoreText = fmt.Sprintf("%s %s Loading more items...", cursor, m.spinner.View())
					if m.cursor == loadMoreIdx {
						loadMoreText = selectedStyle.Render(loadMoreText)
					} else {
						loadMoreText = loadMoreStyle.Render(loadMoreText)
					}
				} else {
					if remaining > 30 {
						loadMoreText = fmt.Sprintf("%s Load More (+30)", cursor)
					} else {
						loadMoreText = fmt.Sprintf("%s Load All (+%d)", cursor, remaining)
					}

					if m.cursor == loadMoreIdx {
						loadMoreText = selectedStyle.Render(loadMoreText)
					} else {
						loadMoreText = loadMoreStyle.Render(loadMoreText)
					}
				}

				content.WriteString(loadMoreText + "\n")
			}
			continue
		}

		treeItem := treeItems[i]
		isSelected := m.cursor == i

		cursor := " "
		if isSelected {
			cursor = "❯"
		}

		// Get tree drawing prefix and color it
		var treePrefix string
		if isSelected {
			treePrefix = edgeStyleSelected.Render(getTreePrefix(treeItem))
		} else {
			treePrefix = edgeStyle.Render(getTreePrefix(treeItem))
		}

		// Get work item type icon
		icon := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
		var styledIcon string
		if isSelected {
			styledIcon = iconStyleSelected.Render(icon)
		} else {
			styledIcon = iconStyle.Render(icon)
		}

		// Get state category to determine styling
		category := m.getStateCategory(treeItem.WorkItem.State)

		// Choose state style based on category and selection
		var stateStyle lipgloss.Style
		var itemTitleStyle lipgloss.Style

		if isSelected {
			// When selected, use bright colors with selected background
			stateStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)
			itemTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)
		} else {
			// Normal styling when not selected
			switch category {
			case "Proposed":
				stateStyle = proposedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248")) // Light gray
			case "InProgress":
				stateStyle = inProgressStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // White
			case "Completed":
				stateStyle = completedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("243"))
			case "Removed":
				stateStyle = removedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("241"))
			default:
				stateStyle = proposedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			}

			// Make title bold if this is a parent (has children)
			if len(treeItem.WorkItem.Children) > 0 {
				itemTitleStyle = itemTitleStyle.Bold(true)
			}
		}

		// Apply the styling
		taskTitle := itemTitleStyle.Render(treeItem.WorkItem.Title)
		state := stateStyle.Render(treeItem.WorkItem.State)

		// Build the line with icon
		var line string
		if isSelected {
			// Apply background to cursor and spacing too
			cursorStyled := selectedStyle.Render(cursor)
			spacer := selectedStyle.Render(" ")
			line = fmt.Sprintf("%s%s%s%s%s%s%s",
				cursorStyled,
				spacer,
				treePrefix,
				styledIcon,
				spacer,
				taskTitle,
				spacer+state)
		} else {
			line = fmt.Sprintf("%s %s%s %s %s",
				cursor,
				treePrefix,
				styledIcon,
				taskTitle,
				state)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	keybindings := "tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate • ctrl+u/d or pgup/pgdn: page up/down\nenter: details • o: open in browser • /: filter • f: find • r: refresh • ?: help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderDetailView() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	var content strings.Builder

	// Title bar
	titleText := fmt.Sprintf("Work Item Details")
	content.WriteString(m.renderTitleBar(titleText))

	// Render the card
	content.WriteString(m.buildDetailContent())
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "←/h/esc: back • r: refresh • e: edit • o: open in browser • s: change state • ?: help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderStatePickerView() string {
	var content strings.Builder

	// Title bar
	titleText := "Select New State"
	content.WriteString(m.renderTitleBar(titleText))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	if m.selectedTask != nil {
		content.WriteString(fmt.Sprintf("Current state: %s\n\n", m.selectedTask.State))
	}

	for i, state := range m.availableStates {
		cursor := " "
		if m.stateCursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, state)

		if m.stateCursor == i {
			line = selectedStyle.Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	keybindings := "↑/↓ or j/k: navigate • ctrl+u/d or pgup/pgdn: page up/down • enter: select • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderFilterView() string {
	var content strings.Builder

	// Title bar
	titleText := "Filter"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.filterInput.View() + "\n\n")

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	// State styles based on category
	proposedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	inProgressStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	completedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	removedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	treeItems := m.getVisibleTreeItems()
	resultCount := len(treeItems)

	// Show result count
	currentTasks := m.getCurrentTasks()
	if m.filterInput.Value() != "" {
		content.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d", resultCount, len(currentTasks))) + "\n\n")
	} else {
		content.WriteString(dimStyle.Render(fmt.Sprintf("  %d items", len(currentTasks))) + "\n\n")
	}

	// Display results (limit to 15 visible items for performance)
	const maxVisible = 15
	startIdx := 0
	endIdx := min(resultCount, maxVisible)

	// Adjust visible window if cursor is out of view
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
		endIdx = m.cursor + 1
	}

	// Style for tree edges with vibrant color
	edgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	edgeStyleSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62"))

	// Icon style
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	iconStyleSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62"))

	for i := startIdx; i < endIdx && i < resultCount; i++ {
		treeItem := treeItems[i]
		isSelected := m.cursor == i

		cursor := "  "
		if isSelected {
			cursor = "❯ "
		}

		// Get tree drawing prefix and color it
		var treePrefix string
		if isSelected {
			treePrefix = edgeStyleSelected.Render(getTreePrefix(treeItem))
		} else {
			treePrefix = edgeStyle.Render(getTreePrefix(treeItem))
		}

		// Get work item type icon
		icon := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
		var styledIcon string
		if isSelected {
			styledIcon = iconStyleSelected.Render(icon)
		} else {
			styledIcon = iconStyle.Render(icon)
		}

		// Get state category to determine styling
		category := m.getStateCategory(treeItem.WorkItem.State)

		// Choose state style based on category and selection
		var stateStyle lipgloss.Style
		var itemTitleStyle lipgloss.Style

		if isSelected {
			// When selected, use bright colors with selected background
			stateStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)
			itemTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)
		} else {
			// Normal styling when not selected
			switch category {
			case "Proposed":
				stateStyle = proposedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			case "InProgress":
				stateStyle = inProgressStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
			case "Completed":
				stateStyle = completedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("243"))
			case "Removed":
				stateStyle = removedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("241"))
			default:
				stateStyle = proposedStateStyle
				itemTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
			}

			// Make title bold if this is a parent
			if len(treeItem.WorkItem.Children) > 0 {
				itemTitleStyle = itemTitleStyle.Bold(true)
			}
		}

		// Apply the styling
		taskTitle := itemTitleStyle.Render(treeItem.WorkItem.Title)
		state := stateStyle.Render(treeItem.WorkItem.State)

		// Build the line with icon
		var line string
		if isSelected {
			// Apply background to cursor and spacing too
			cursorStyled := selectedStyle.Render(cursor)
			spacer := selectedStyle.Render(" ")
			line = fmt.Sprintf("%s%s%s%s%s%s",
				cursorStyled,
				treePrefix,
				styledIcon,
				spacer,
				taskTitle,
				spacer+state)
		} else {
			line = fmt.Sprintf("%s%s%s %s %s",
				cursor,
				treePrefix,
				styledIcon,
				taskTitle,
				state)
		}

		content.WriteString(line + "\n")
	}

	// Show scroll indicator if there are more items
	if resultCount > maxVisible {
		if endIdx < resultCount {
			content.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more ...", resultCount-endIdx)) + "\n")
		}
	}

	// Footer with keybindings
	keybindings := "↑/↓ or ctrl+j/k: navigate • ctrl+u/d or pgup/pgdn: page up/down • enter: open detail • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderFindView() string {
	var content strings.Builder

	// Title bar
	titleText := "Find Work Items"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.findInput.View() + "\n\n")

	// Note about query behavior
	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	content.WriteString(noteStyle.Render("Note: Queries all work items assigned to @Me") + "\n")

	// Footer with keybindings
	keybindings := "enter: apply find • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderHelpView() string {
	var content strings.Builder

	// Title bar
	titleText := "Keybindings Help"
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(20)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	// Global keybindings
	content.WriteString(sectionStyle.Render("Global") + "\n")
	content.WriteString(keyStyle.Render("?") + descStyle.Render("Show/hide this help") + "\n")
	content.WriteString(keyStyle.Render("q, ctrl+c") + descStyle.Render("Quit application") + "\n")
	content.WriteString(keyStyle.Render("ctrl+u/d, pgup/pgdn") + descStyle.Render("Jump half page up/down (works in all views)") + "\n")
	content.WriteString(keyStyle.Render("r") + descStyle.Render("Refresh (all data in list, single item in detail)") + "\n")
	content.WriteString(keyStyle.Render("o") + descStyle.Render("Open current item in browser") + "\n\n")

	// List view keybindings
	content.WriteString(sectionStyle.Render("List View") + "\n")
	content.WriteString(keyStyle.Render("tab") + descStyle.Render("Cycle through tabs (sprint or backlog)") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate up/down") + "\n")
	content.WriteString(keyStyle.Render("→/l, enter") + descStyle.Render("Open item details") + "\n")
	content.WriteString(keyStyle.Render("/") + descStyle.Render("Filter items in current list") + "\n")
	content.WriteString(keyStyle.Render("f") + descStyle.Render("Find items with dedicated query") + "\n\n")

	// Detail view keybindings
	content.WriteString(sectionStyle.Render("Detail View") + "\n")
	content.WriteString(keyStyle.Render("←/h, esc, backspace") + descStyle.Render("Back to list") + "\n")
	content.WriteString(keyStyle.Render("e") + descStyle.Render("Edit item") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change item state") + "\n\n")

	// State picker view keybindings
	content.WriteString(sectionStyle.Render("State Picker") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate states") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Select state") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel") + "\n\n")

	// Filter view keybindings
	content.WriteString(sectionStyle.Render("Filter View") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel filter") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Open selected item") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, ctrl+j/k") + descStyle.Render("Navigate results") + "\n\n")

	// Footer with keybindings
	keybindings := "?: close help • esc: close help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderEditView() string {
	var content strings.Builder

	// Title bar
	titleText := "Edit Work Item"
	if m.selectedTask != nil {
		titleText = fmt.Sprintf("Edit Work Item #%d", m.selectedTask.ID)
	}
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(15)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	sectionStyle := lipgloss.NewStyle().
		MarginTop(1).
		MarginBottom(1)

	// Show loading spinner if saving
	if m.loading {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Saving changes..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	// Title field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Title:") + "\n" + m.editTitleInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 0 {
		content.WriteString(helpStyle.Render("  Enter the work item title") + "\n")
	}
	content.WriteString("\n")

	// Description field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Description:") + "\n" + m.editDescriptionInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 1 {
		content.WriteString(helpStyle.Render("  Multi-line text editor (HTML will be stripped)") + "\n")
	}
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "tab/shift+tab: switch field • ctrl+s: save • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func openInBrowser(orgURL, project string, workItemID int) error {
	// Clean up org URL
	orgURL = strings.TrimSuffix(orgURL, "/")

	url := fmt.Sprintf("%s/%s/_workitems/edit/%d", orgURL, project, workItemID)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // linux and others
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
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
