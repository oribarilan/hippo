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

type workItemCreatedMsg struct {
	workItem *WorkItem
	err      error
}

type workItemDeletedMsg struct {
	workItemID int
	err        error
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
	forceReload    bool // Force reload even if sprints already exist
}

func loadTasks(client *AzureDevOpsClient) tea.Cmd {
	return loadTasksForSprint(client, nil, "", defaultLoadLimit, nil)
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
	return loadTasksForSprint(client, nil, sprintPath, defaultLoadLimit, &tabCopy)
}

func loadTasksForBacklogTab(client *AzureDevOpsClient, tab backlogTab, currentSprintPath string) tea.Cmd {
	return func() tea.Msg {
		var tasks []WorkItem
		var err error
		var totalCount int

		switch tab {
		case recentBacklog:
			tasks, err = client.GetRecentBacklogItems(defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItems(currentSprintPath, defaultLoadLimit)
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
			tasks, err = client.GetRecentBacklogItemsExcluding(excludeIDs, defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItemsExcluding(excludeIDs, currentSprintPath, defaultLoadLimit)
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
	return loadSprintsWithReload(client, false)
}

func loadSprintsWithReload(client *AzureDevOpsClient, forceReload bool) tea.Cmd {
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
			forceReload:    forceReload,
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

func createWorkItem(client *AzureDevOpsClient, title string, workItemType string, iterationPath string, parentID *int, areaPath string) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.CreateWorkItem(title, workItemType, iterationPath, parentID, areaPath)
		return workItemCreatedMsg{workItem: workItem, err: err}
	}
}

func deleteWorkItem(client *AzureDevOpsClient, workItemID int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteWorkItem(workItemID)
		return workItemDeletedMsg{workItemID: workItemID, err: err}
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

		// Handle error view
		if m.state == errorView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.statusMessage = ""
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
				// Clear filter in current list
				if list := m.getCurrentList(); list != nil {
					list.filterActive = false
					list.filteredTasks = nil
				}
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
				// Return to appropriate view
				if len(m.selectedItems) > 0 {
					// Was batch operation, return to list
					m.state = listView
				} else {
					// Was single item, return to detail
					m.state = detailView
				}
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
				newState := m.availableStates[m.stateCursor]

				// Check if batch operation or single item
				if len(m.selectedItems) > 0 && m.client != nil {
					// Batch state update
					m.loading = true
					count := len(m.selectedItems)
					m.batchOperationCount = count // Track batch operations
					m.statusMessage = fmt.Sprintf("Updating %d items to %s...", count, newState)
					m.state = listView

					var updateCmds []tea.Cmd
					for itemID := range m.selectedItems {
						updateCmds = append(updateCmds, updateWorkItemState(m.client, itemID, newState))
					}

					// Clear selection after starting update
					m.selectedItems = make(map[int]bool)
					updateCmds = append(updateCmds, m.spinner.Tick)
					return m, tea.Batch(updateCmds...)
				} else if m.selectedTask != nil && m.client != nil {
					// Single item state update
					m.loading = true
					m.batchOperationCount = 1 // Single operation
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

		// Handle create view
		if m.state == createView {
			switch msg.String() {
			case "esc":
				// Cancel creation and return to list view
				m.state = listView
				m.createInput.SetValue("")
				return m, nil
			case "enter":
				// Show help hint instead of submitting
				m.statusMessage = "Use ctrl+s to save, esc to cancel"
				return m, nil
			case "ctrl+s":
				// Save new work item
				title := strings.TrimSpace(m.createInput.Value())
				if title == "" {
					m.statusMessage = "Title cannot be empty"
					return m, nil
				}

				if m.client != nil {
					// Get current sprint path
					var iterationPath string
					if m.currentMode == sprintMode {
						sprint := m.sprints[m.currentTab]
						if sprint != nil {
							iterationPath = sprint.Path
						}
					}

					// Get area path from parent if creating a child, otherwise from any item in list
					var areaPath string
					currentTasks := m.getCurrentTasks()

					if m.createParentID != nil {
						// Find parent in current tasks to get its area path
						for _, task := range currentTasks {
							if task.ID == *m.createParentID {
								areaPath = task.AreaPath
								break
							}
						}
					} else if len(currentTasks) > 0 {
						// No parent - get area path from first item in list (arbitrary but consistent)
						areaPath = currentTasks[0].AreaPath
					}
					// If still no area path found (empty list), leave empty to use project default

					m.loading = true
					m.statusMessage = "Creating work item..."
					return m, tea.Batch(
						createWorkItem(m.client, title, "Task", iterationPath, m.createParentID, areaPath),
						m.spinner.Tick,
					)
				}
				return m, nil
			default:
				// Update the input field
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(msg)
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
				currentList := m.getCurrentList()
				if currentList != nil && !currentList.attempted && m.client != nil {
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
					m.sprintLists = make(map[sprintTab]*WorkItemList)
					return m, tea.Batch(loadSprintsWithReload(m.client, true), m.spinner.Tick)
				} else {
					// Clear backlog data
					m.backlogLists = make(map[backlogTab]*WorkItemList)
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
			// Change state - in detail view or list view (for batch)
			if m.state == detailView && m.selectedTask != nil && m.client != nil {
				// Single item state change from detail view
				m.loading = true
				m.statusMessage = "Loading states..."
				return m, tea.Batch(
					loadWorkItemStates(m.client, m.selectedTask.WorkItemType),
					m.spinner.Tick,
				)
			} else if m.state == listView && len(m.selectedItems) > 0 && m.client != nil {
				// Batch state change from list view
				m.loading = true
				m.statusMessage = "Loading states..."
				// Use "Task" as default work item type for batch operations
				return m, tea.Batch(
					loadWorkItemStates(m.client, "Task"),
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

		case "i", "a":
			// Insert (i) or append (a) new work item - only in list view
			if m.state != listView {
				return m, nil
			}

			treeItems := m.getVisibleTreeItems()
			if len(treeItems) == 0 {
				// Empty list - create first item at root level
				m.createInsertPos = 0
				m.createAfter = false
				m.createParentID = nil
				m.createDepth = 0
				m.createIsLast = []bool{}
			} else if m.cursor >= len(treeItems) {
				// On "Load More" item - do nothing
				return m, nil
			} else {
				item := treeItems[m.cursor]
				m.createAfter = (msg.String() == "a")

				if m.createAfter {
					// Append after - check if current item has children or can have children
					// If current item is a parent (has children or could be one), create as first child
					if len(item.WorkItem.Children) > 0 {
						// Has children - insert as first child (right after parent)
						m.createInsertPos = m.cursor + 1
						m.createDepth = item.Depth + 1
						m.createParentID = &item.WorkItem.ID
						// Append false to IsLast (not last in parent's IsLast chain, plus new child level)
						m.createIsLast = append([]bool{}, item.IsLast...)
						m.createIsLast = append(m.createIsLast, false) // First child, not last
					} else {
						// No children - create as sibling after the subtree
						m.createInsertPos = getPositionAfterSubtree(treeItems, m.cursor)
						// Determine parent and depth based on position after subtree
						if m.createInsertPos < len(treeItems) {
							// There's an item after the subtree - use same depth as cursor item (sibling)
							m.createDepth = item.Depth
							m.createParentID = getParentIDForTreeItem(item)
							m.createIsLast = append([]bool{}, item.IsLast...)
						} else {
							// Appending at end of list - same depth as cursor item
							m.createDepth = item.Depth
							m.createParentID = getParentIDForTreeItem(item)
							m.createIsLast = append([]bool{}, item.IsLast...)
							if len(m.createIsLast) > 0 {
								m.createIsLast[len(m.createIsLast)-1] = true // Mark as last
							}
						}
					}
				} else {
					// Insert before - same depth and parent as current item
					m.createInsertPos = m.cursor
					m.createDepth = item.Depth
					m.createParentID = getParentIDForTreeItem(item)
					m.createIsLast = append([]bool{}, item.IsLast...)
				}
			}

			m.createInput.SetValue("")
			m.createInput.Focus()
			m.state = createView
			return m, nil

		case "d":
			// Delete work item(s) - only in list view
			if m.state != listView {
				return m, nil
			}

			// Check if we have batch selection
			if len(m.selectedItems) > 0 {
				// Batch delete - just set flag, will be handled in confirmation
				m.state = deleteConfirmView
				return m, nil
			}

			// Single delete
			treeItems := m.getVisibleTreeItems()
			if len(treeItems) == 0 || m.cursor >= len(treeItems) {
				// Empty list or on "Load More" - do nothing
				return m, nil
			}

			// Store the item to delete and show confirmation
			item := treeItems[m.cursor].WorkItem
			m.deleteItemID = item.ID
			m.deleteItemTitle = item.Title
			m.state = deleteConfirmView
			return m, nil
		}

		// Handle delete confirmation view
		if m.state == deleteConfirmView {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if m.client != nil {
					m.loading = true
					m.state = listView

					// Check if batch delete or single delete
					if len(m.selectedItems) > 0 {
						// Batch delete
						var deleteCmds []tea.Cmd
						count := len(m.selectedItems)
						m.batchOperationCount = count // Track batch operations
						m.statusMessage = fmt.Sprintf("Deleting %d work items...", count)

						for itemID := range m.selectedItems {
							deleteCmds = append(deleteCmds, deleteWorkItem(m.client, itemID))
						}

						// Clear selection after starting delete
						m.selectedItems = make(map[int]bool)
						deleteCmds = append(deleteCmds, m.spinner.Tick)
						return m, tea.Batch(deleteCmds...)
					} else {
						// Single delete
						m.batchOperationCount = 1 // Single operation
						m.statusMessage = "Deleting work item..."
						return m, tea.Batch(deleteWorkItem(m.client, m.deleteItemID), m.spinner.Tick)
					}
				}
				m.state = listView
				return m, nil
			case "n", "N", "esc":
				// Cancel delete
				m.state = listView
				return m, nil
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
				// Clear selections when switching tabs
				m.selectedItems = make(map[int]bool)

				if m.currentMode == sprintMode {
					m.currentTab = (m.currentTab + 1) % 3
					// Restore cursor/scroll from the new tab's list
					if list := m.getCurrentList(); list != nil {
						m.cursor = list.cursor
						m.scrollOffset = list.scrollOffset
					} else {
						m.cursor = 0
						m.scrollOffset = 0
					}
					// Load sprint data if not attempted yet
					currentList := m.getCurrentList()
					sprint := m.sprints[m.currentTab]
					if currentList != nil && !currentList.attempted && sprint != nil && m.client != nil {
						m.loading = true
						tab := m.currentTab
						return m, tea.Batch(loadTasksForSprint(m.client, nil, sprint.Path, defaultLoadLimit, &tab), m.spinner.Tick)
					}
				} else if m.currentMode == backlogMode {
					m.currentBacklogTab = (m.currentBacklogTab + 1) % 2
					// Restore cursor/scroll from the new tab's list
					if list := m.getCurrentList(); list != nil {
						m.cursor = list.cursor
						m.scrollOffset = list.scrollOffset
					} else {
						m.cursor = 0
						m.scrollOffset = 0
					}
					// Load backlog data if not attempted yet
					currentList := m.getCurrentList()
					if currentList != nil && !currentList.attempted && m.client != nil {
						m.loading = true
						tab := m.currentBacklogTab
						return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
					}
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					m.adjustScrollOffset()
					// Also update current list's cursor
					if list := m.getCurrentList(); list != nil {
						list.cursor = m.cursor
						list.scrollOffset = m.scrollOffset
					}
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
					// Also update current list's cursor
					if list := m.getCurrentList(); list != nil {
						list.cursor = m.cursor
						list.scrollOffset = m.scrollOffset
					}
				}
			case "ctrl+u", "pgup":
				// Jump up half page
				m.cursor = max(0, m.cursor-10)
				m.adjustScrollOffset()
				// Also update current list's cursor
				if list := m.getCurrentList(); list != nil {
					list.cursor = m.cursor
					list.scrollOffset = m.scrollOffset
				}
			case "ctrl+d", "pgdown":
				// Jump down half page
				treeItems := m.getVisibleTreeItems()
				maxCursor := len(treeItems) - 1
				if m.hasMoreItems() {
					maxCursor = len(treeItems)
				}
				m.cursor = min(maxCursor, m.cursor+10)
				m.adjustScrollOffset()
				// Also update current list's cursor
				if list := m.getCurrentList(); list != nil {
					list.cursor = m.cursor
					list.scrollOffset = m.scrollOffset
				}
			case " ":
				// Toggle selection for current item
				treeItems := m.getVisibleTreeItems()
				if len(treeItems) > 0 && m.cursor < len(treeItems) {
					itemID := treeItems[m.cursor].WorkItem.ID
					if m.selectedItems[itemID] {
						delete(m.selectedItems, itemID)
					} else {
						m.selectedItems[itemID] = true
					}
				}
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
							list := m.getCurrentList()
							if list != nil {
								for _, task := range list.tasks {
									if task.IterationPath == sprintPath {
										excludeIDs = append(excludeIDs, task.ID)
									}
								}
							}

							tab := m.currentTab
							return m, tea.Batch(loadTasksForSprint(m.client, excludeIDs, sprintPath, defaultLoadLimit, &tab), m.spinner.Tick)
						} else if m.currentMode == backlogMode {
							// Load more items for backlog mode
							// Collect all currently loaded IDs in this backlog tab to exclude
							excludeIDs := make([]int, 0)
							list := m.getCurrentList()
							if list != nil {
								for _, task := range list.tasks {
									excludeIDs = append(excludeIDs, task.ID)
								}
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
			// Add context to error message based on which tab failed
			var errorContext string
			if msg.forTab != nil {
				tabNames := map[sprintTab]string{
					previousSprint: "previous sprint",
					currentSprint:  "current sprint",
					nextSprint:     "next sprint",
				}
				if tabName, ok := tabNames[*msg.forTab]; ok {
					errorContext = fmt.Sprintf(" (%s)", tabName)
				}
			} else if msg.forBacklogTab != nil {
				tabNames := map[backlogTab]string{
					recentBacklog: "recent backlog",
					abandonedWork: "abandoned work",
				}
				if tabName, ok := tabNames[*msg.forBacklogTab]; ok {
					errorContext = fmt.Sprintf(" (%s)", tabName)
				}
			}

			// Wrap error with context if available
			if errorContext != "" {
				m.err = fmt.Errorf("failed to load%s: %w", errorContext, msg.err)
			} else {
				m.err = msg.err
			}

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

				// Ensure list exists for this tab
				if m.sprintLists[targetTab] == nil {
					m.sprintLists[targetTab] = &WorkItemList{}
				}
				list := m.sprintLists[targetTab]

				if msg.append {
					// Append new tasks to existing ones (load more scenario)
					list.appendTasks(msg.tasks)
					list.totalCount = msg.totalCount

					m.setActionLog(fmt.Sprintf("Loaded %d more items", len(msg.tasks)))
					m.loading = false
					m.loadingMore = false
				} else {
					// Initial load or replace
					list.replaceTasks(msg.tasks, msg.totalCount)

					if targetTab == m.currentTab && m.currentMode == sprintMode {
						// Sync cursor/scroll from list
						m.cursor = list.cursor
						m.scrollOffset = list.scrollOffset
					}
					m.statusMessage = ""

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

				// Ensure list exists for this tab
				if m.backlogLists[targetTab] == nil {
					m.backlogLists[targetTab] = &WorkItemList{}
				}
				list := m.backlogLists[targetTab]

				if msg.append {
					// Append new tasks to existing ones (load more scenario)
					list.appendTasks(msg.tasks)
					list.totalCount = msg.totalCount

					m.setActionLog(fmt.Sprintf("Loaded %d more items", len(msg.tasks)))
					m.loading = false
					m.loadingMore = false
				} else {
					// Store tasks for specific backlog tab
					list.replaceTasks(msg.tasks, msg.totalCount)

					if targetTab == m.currentBacklogTab && m.currentMode == backlogMode {
						// Sync cursor/scroll from list
						m.cursor = list.cursor
						m.scrollOffset = list.scrollOffset
					}
					m.statusMessage = ""

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
		if msg.err != nil {
			// Show detailed error
			m.loading = false
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error updating state: %v", msg.err))
			m.batchOperationCount = 0 // Reset on error
			m.state = listView
			m.stateCursor = 0
		} else {
			// Success! Decrement counter
			if m.batchOperationCount > 0 {
				m.batchOperationCount--
			}

			// Only refresh when all operations are complete
			if m.batchOperationCount == 0 {
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
				m.statusMessage = "State updated successfully!"
				if taskTitle != "" && oldState != "" && newState != "" {
					m.setActionLog(fmt.Sprintf("Updated \"%s\": %s → %s", taskTitle, oldState, newState))
				} else {
					m.setActionLog("State updated successfully")
				}

				// Refresh the list
				if m.client != nil {
					m.loading = true
					m.statusMessage = "Refreshing list..."
					if m.currentMode == sprintMode {
						// Clear sprint data and reload
						m.sprintLists = make(map[sprintTab]*WorkItemList)
						return m, tea.Batch(loadSprintsWithReload(m.client, true), m.spinner.Tick)
					} else {
						// Clear backlog data and reload current tab
						m.backlogLists = make(map[backlogTab]*WorkItemList)
						tab := m.currentBacklogTab
						return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
					}
				}
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

			// Invalidate tree cache since task was updated
			if list := m.getCurrentList(); list != nil {
				list.invalidateTreeCache()
			}

			m.setActionLog(fmt.Sprintf("Refreshed #%d", msg.workItem.ID))
		}

	case workItemCreatedMsg:
		if msg.err != nil {
			// Show detailed error in error view
			m.loading = false
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error creating work item: %v", msg.err))
			// Switch to error view to show full details
			m.state = errorView
		} else if msg.workItem != nil {
			// Success! Store the created item ID and continue with spinner
			m.createdItemID = msg.workItem.ID
			m.statusMessage = "Refreshing list..."
			m.setActionLog(fmt.Sprintf("Created #%d: %s", msg.workItem.ID, msg.workItem.Title))

			// Return to list view and trigger refresh (keeping spinner going)
			m.state = listView

			// Trigger refresh based on current mode
			if m.client != nil {
				if m.currentMode == sprintMode {
					// Clear sprint data and reload
					m.sprintLists = make(map[sprintTab]*WorkItemList)
					return m, tea.Batch(loadSprintsWithReload(m.client, true), m.spinner.Tick)
				} else {
					// Clear backlog data and reload current tab
					m.backlogLists = make(map[backlogTab]*WorkItemList)
					tab := m.currentBacklogTab
					return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
				}
			}
		}
		// If we get here without returning, clear loading state
		m.loading = false
		m.statusMessage = ""

	case workItemDeletedMsg:
		if msg.err != nil {
			// Show detailed error
			m.loading = false
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			m.setActionLog(fmt.Sprintf("Error deleting work item: %v", msg.err))
			m.batchOperationCount = 0 // Reset on error
		} else {
			// Success! Decrement counter
			if m.batchOperationCount > 0 {
				m.batchOperationCount--
			}

			// Only refresh when all operations are complete
			if m.batchOperationCount == 0 {
				m.statusMessage = "Refreshing list..."
				m.setActionLog(fmt.Sprintf("Deleted work item(s)"))

				// Trigger refresh to update the list
				if m.client != nil {
					if m.currentMode == sprintMode {
						// Clear sprint data and reload
						m.sprintLists = make(map[sprintTab]*WorkItemList)
						return m, tea.Batch(loadSprintsWithReload(m.client, true), m.spinner.Tick)
					} else {
						// Clear backlog data and reload current tab
						m.backlogLists = make(map[backlogTab]*WorkItemList)
						tab := m.currentBacklogTab
						return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick)
					}
				}
				m.loading = false
				m.statusMessage = ""
			}
		}

	case sprintsLoadedMsg:
		// Store client first
		if msg.client != nil {
			m.client = msg.client
		}

		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading sprints: %v", msg.err)
		} else {
			needsReload := len(m.sprints) == 0 || msg.forceReload // First time loading sprints OR forced reload

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

func (m *model) filterSearch() {
	query := strings.ToLower(m.filterInput.Value())
	list := m.getCurrentList()
	if list == nil {
		return
	}

	if query == "" {
		list.filteredTasks = nil
		list.filterActive = false
		// Also sync model-level filter state
		m.filteredTasks = nil
		// Invalidate cache when filter is cleared
		list.invalidateTreeCache()
		return
	}

	var filtered []WorkItem
	for _, task := range list.tasks {
		if strings.Contains(strings.ToLower(task.Title), query) ||
			strings.Contains(fmt.Sprintf("%d", task.ID), query) {
			filtered = append(filtered, task)
		}
	}
	list.filteredTasks = filtered
	list.filterActive = true
	// Also sync model-level filter state for backward compatibility
	m.filteredTasks = filtered
	// Invalidate cache when filter changes
	list.invalidateTreeCache()
}

func (m model) getVisibleTasks() []WorkItem {
	list := m.getCurrentList()
	if list == nil {
		return []WorkItem{}
	}

	tasks := list.getVisibleTasks()

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
// Uses caching to avoid rebuilding the tree on every render
func (m model) getVisibleTreeItems() []TreeItem {
	list := m.getCurrentList()
	if list == nil {
		return []TreeItem{}
	}

	visibleTasks := m.getVisibleTasks()
	if len(visibleTasks) == 0 {
		return []TreeItem{}
	}

	// Check if we have a valid cache
	// Cache is valid if it exists and the number of visible tasks hasn't changed
	if list.treeCache != nil && len(list.treeCache) > 0 {
		// Quick validation: check if task count matches
		if len(visibleTasks) == countTreeItems(list.treeCache) {
			return list.treeCache
		}
	}

	// Cache miss or invalid - rebuild tree structure
	roots := buildTreeStructure(visibleTasks)
	list.treeCache = flattenTree(roots)

	return list.treeCache
}

func (m model) hasMoreItems() bool {
	if list := m.getCurrentList(); list != nil {
		return list.hasMore()
	}
	return false
}

func (m model) getRemainingCount() int {
	if list := m.getCurrentList(); list != nil {
		return list.getRemainingCount()
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

// getCurrentList returns the currently active WorkItemList based on mode and tab
func (m model) getCurrentList() *WorkItemList {
	if m.currentMode == sprintMode {
		if list, ok := m.sprintLists[m.currentTab]; ok {
			return list
		}
		// If list doesn't exist yet, create it
		return &WorkItemList{}
	}
	// Backlog mode
	if list, ok := m.backlogLists[m.currentBacklogTab]; ok {
		return list
	}
	// If list doesn't exist yet, create it
	return &WorkItemList{}
}

// ensureCurrentListExists makes sure the current list is initialized
func (m *model) ensureCurrentListExists() {
	if m.currentMode == sprintMode {
		if _, ok := m.sprintLists[m.currentTab]; !ok {
			m.sprintLists[m.currentTab] = &WorkItemList{}
		}
	} else {
		if _, ok := m.backlogLists[m.currentBacklogTab]; !ok {
			m.backlogLists[m.currentBacklogTab] = &WorkItemList{}
		}
	}
}

// buildDetailContent creates the content string for the detail view
func (m model) buildDetailContent() string {
	if m.selectedTask == nil {
		return ""
	}

	task := m.selectedTask

	// Use card style with dynamic width
	cardStyle := m.styles.Card.Width(m.width - 4)

	// Build the card content
	var cardContent strings.Builder

	// Header with ID and Title
	cardContent.WriteString(m.styles.Header.Render(fmt.Sprintf("#%d - %s", task.ID, task.Title)))
	cardContent.WriteString("\n\n")

	// Parent info if exists
	if parent := m.getParentTask(task); parent != nil {
		cardContent.WriteString(m.styles.Label.Render("Parent:"))
		cardContent.WriteString(m.styles.Value.Render(fmt.Sprintf("#%d - %s", parent.ID, parent.Title)))
		cardContent.WriteString("\n")
	}

	// Basic Info
	cardContent.WriteString(m.styles.Label.Render("Type:"))
	cardContent.WriteString(m.styles.Value.Render(task.WorkItemType))
	cardContent.WriteString("\n")

	cardContent.WriteString(m.styles.Label.Render("State:"))
	cardContent.WriteString(m.styles.Value.Render(task.State))
	cardContent.WriteString("\n")

	if task.AssignedTo != "" {
		cardContent.WriteString(m.styles.Label.Render("Assigned To:"))
		cardContent.WriteString(m.styles.Value.Render(task.AssignedTo))
		cardContent.WriteString("\n")
	}

	if task.Priority > 0 {
		cardContent.WriteString(m.styles.Label.Render("Priority:"))
		cardContent.WriteString(m.styles.Value.Render(fmt.Sprintf("%d", task.Priority)))
		cardContent.WriteString("\n")
	}

	if task.Tags != "" {
		cardContent.WriteString(m.styles.Label.Render("Tags:"))
		cardContent.WriteString(m.styles.Value.Render(task.Tags))
		cardContent.WriteString("\n")
	}

	if task.IterationPath != "" {
		cardContent.WriteString(m.styles.Label.Render("Sprint:"))
		// Extract just the sprint name from the path
		parts := strings.Split(task.IterationPath, "\\")
		sprintName := parts[len(parts)-1]
		cardContent.WriteString(m.styles.Value.Render(sprintName))
		cardContent.WriteString("\n")
	}

	if task.CreatedDate != "" {
		formattedDate := formatDateTime(task.CreatedDate)
		relativeTime := getRelativeTime(task.CreatedDate)
		dateValue := formattedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", formattedDate, relativeTime)
		}
		cardContent.WriteString(m.styles.Label.Render("Created:"))
		cardContent.WriteString(m.styles.Value.Render(dateValue))
		cardContent.WriteString("\n")
	}

	if task.ChangedDate != "" {
		formattedDate := formatDateTime(task.ChangedDate)
		relativeTime := getRelativeTime(task.ChangedDate)
		dateValue := formattedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", formattedDate, relativeTime)
		}
		cardContent.WriteString(m.styles.Label.Render("Last Updated:"))
		cardContent.WriteString(m.styles.Value.Render(dateValue))
		cardContent.WriteString("\n")
	}

	// Description Section
	if task.Description != "" {
		cardContent.WriteString("\n")
		cardContent.WriteString(m.styles.Section.Render("Description"))
		cardContent.WriteString("\n")
		cardContent.WriteString(m.styles.Description.Render(task.Description))
	}

	// Comments / Discussion Section
	if task.Comments != "" {
		cardContent.WriteString("\n")
		cardContent.WriteString(m.styles.Section.Render("Comments / Discussion"))
		cardContent.WriteString("\n")
		cardContent.WriteString(m.styles.Description.Render(task.Comments))
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
	if list := m.getCurrentList(); list != nil {
		return list.tasks
	}
	return []WorkItem{}
}

// setCurrentTasks sets the task list for the current mode
func (m *model) setCurrentTasks(tasks []WorkItem) {
	m.ensureCurrentListExists()
	if list := m.getCurrentList(); list != nil {
		list.tasks = tasks
		list.loaded = len(tasks)
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

	timestamp := m.lastActionTime.Format("15:04:05")
	return m.styles.Log.Render(fmt.Sprintf("[%s] %s", timestamp, m.lastActionLog))
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
		Background(lipgloss.Color(ColorPurple)).
		Foreground(lipgloss.Color(ColorWhite)).
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
	separatorStyle := m.styles.Separator.Width(m.width)
	separator := separatorStyle.Render(strings.Repeat("─", m.width))

	footer.WriteString(separator + "\n")
	footer.WriteString(m.styles.Help.Render(keybindings))

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
	case createView:
		return m.renderCreateView()
	case errorView:
		return m.renderErrorView()
	case deleteConfirmView:
		return m.renderDeleteConfirmView()
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

	// Render mode selector
	modes := []string{}
	if m.currentMode == sprintMode {
		modes = append(modes, m.styles.ActiveMode.Render("[1] Sprint"))
		modes = append(modes, m.styles.InactiveMode.Render("[2] Backlog"))
	} else {
		modes = append(modes, m.styles.InactiveMode.Render("[1] Sprint"))
		modes = append(modes, m.styles.ActiveMode.Render("[2] Backlog"))
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
			tabs = append(tabs, m.styles.ActiveTab.Render(prevLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(prevLabel))
		}

		currLabel := "Current Sprint"
		if sprint := m.sprints[currentSprint]; sprint != nil {
			currLabel = sprint.Name
		}
		if m.currentTab == currentSprint {
			tabs = append(tabs, m.styles.ActiveTab.Render(currLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(currLabel))
		}

		nextLabel := "Next Sprint"
		if sprint := m.sprints[nextSprint]; sprint != nil {
			nextLabel = sprint.Name
		}
		if m.currentTab == nextSprint {
			tabs = append(tabs, m.styles.ActiveTab.Render(nextLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(nextLabel))
		}
	} else if m.currentMode == backlogMode {
		// Backlog mode tabs
		if m.currentBacklogTab == recentBacklog {
			tabs = append(tabs, m.styles.ActiveTab.Render("Recent Backlog"))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render("Recent Backlog"))
		}

		if m.currentBacklogTab == abandonedWork {
			tabs = append(tabs, m.styles.ActiveTab.Render("Abandoned Work"))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render("Abandoned Work"))
		}
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// Show tab hint
	if hint := m.getTabHint(); hint != "" {
		content.WriteString(m.styles.Hint.Render(hint) + "\n\n")
	}

	if m.statusMessage != "" {
		content.WriteString(m.styles.StatusMsg.Render(m.statusMessage) + "\n\n")
	}

	// Show loader if loading (but not loadingMore - that shows inline)
	if m.loading && !m.loadingMore {
		content.WriteString(m.styles.Loader.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate\nenter: details • o: open in browser • /: search • f: filter • r: refresh • q: quit"
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()
	if len(treeItems) == 0 {
		content.WriteString("  No tasks found.\n")
	}

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
						loadMoreText = m.styles.Selected.Render(loadMoreText)
					} else {
						loadMoreText = m.styles.LoadMore.Render(loadMoreText)
					}
				} else {
					if remaining > 30 {
						loadMoreText = fmt.Sprintf("%s Load More (+30)", cursor)
					} else {
						loadMoreText = fmt.Sprintf("%s Load All (+%d)", cursor, remaining)
					}

					if m.cursor == loadMoreIdx {
						loadMoreText = m.styles.Selected.Render(loadMoreText)
					} else {
						loadMoreText = m.styles.LoadMore.Render(loadMoreText)
					}
				}

				content.WriteString(loadMoreText + "\n")
			}
			continue
		}

		treeItem := treeItems[i]
		isSelected := m.cursor == i
		isBatchSelected := m.selectedItems[treeItem.WorkItem.ID]

		cursor := " "
		if isSelected {
			cursor = "❯"
		}

		// Orange bar for batch selected items - thicker bar
		batchIndicator := "  "
		if isBatchSelected {
			batchIndicator = m.styles.BatchIndicator.Render("█ ") // Block character for thicker bar
		}

		// Get tree drawing prefix and color it
		var treePrefix string
		if isSelected {
			treePrefix = m.styles.TreeEdgeSelected.Render(getTreePrefix(treeItem))
		} else {
			treePrefix = m.styles.TreeEdge.Render(getTreePrefix(treeItem))
		}

		// Get work item type icon
		icon := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
		var styledIcon string
		if isSelected {
			styledIcon = m.styles.IconSelected.Render(icon)
		} else {
			styledIcon = m.styles.Icon.Render(icon)
		}

		// Get state category to determine styling
		category := m.getStateCategory(treeItem.WorkItem.State)

		// Get styles based on category and selection
		stateStyle := m.styles.GetStateStyle(category, isSelected)
		itemTitleStyle := m.styles.GetItemTitleStyle(category, isSelected, len(treeItem.WorkItem.Children) > 0)

		// Apply the styling
		taskTitle := itemTitleStyle.Render(treeItem.WorkItem.Title)
		state := stateStyle.Render(treeItem.WorkItem.State)

		// Build the line with icon
		var line string
		if isSelected {
			// Apply background to cursor and spacing too
			cursorStyled := m.styles.Selected.Render(cursor)
			spacer := m.styles.Selected.Render(" ")
			line = fmt.Sprintf("%s%s%s%s%s%s%s%s",
				batchIndicator,
				cursorStyled,
				spacer,
				treePrefix,
				styledIcon,
				spacer,
				taskTitle,
				spacer+state)
		} else {
			line = fmt.Sprintf("%s%s %s%s %s %s",
				batchIndicator,
				cursor,
				treePrefix,
				styledIcon,
				taskTitle,
				state)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	batchInfo := ""
	if len(m.selectedItems) > 0 {
		batchInfo = fmt.Sprintf(" • %d items selected", len(m.selectedItems))
	}
	keybindings := fmt.Sprintf("tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate • space: select/deselect%s\ni: insert • a: append • d: delete • s: change state • enter: details • o: open • /: filter • f: find • r: refresh • ?: help • q: quit", batchInfo)
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

	treeItems := m.getVisibleTreeItems()
	resultCount := len(treeItems)

	// Show result count
	currentTasks := m.getCurrentTasks()
	if m.filterInput.Value() != "" {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d/%d", resultCount, len(currentTasks))) + "\n\n")
	} else {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d items", len(currentTasks))) + "\n\n")
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
			treePrefix = m.styles.TreeEdgeSelected.Render(getTreePrefix(treeItem))
		} else {
			treePrefix = m.styles.TreeEdge.Render(getTreePrefix(treeItem))
		}

		// Get work item type icon
		icon := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
		var styledIcon string
		if isSelected {
			styledIcon = m.styles.IconSelected.Render(icon)
		} else {
			styledIcon = m.styles.Icon.Render(icon)
		}

		// Get state category to determine styling
		category := m.getStateCategory(treeItem.WorkItem.State)

		// Get styles based on category and selection
		stateStyle := m.styles.GetStateStyle(category, isSelected)
		itemTitleStyle := m.styles.GetItemTitleStyle(category, isSelected, len(treeItem.WorkItem.Children) > 0)

		// Apply the styling
		taskTitle := itemTitleStyle.Render(treeItem.WorkItem.Title)
		state := stateStyle.Render(treeItem.WorkItem.State)

		// Build the line with icon
		var line string
		if isSelected {
			// Apply background to cursor and spacing too
			cursorStyled := m.styles.Selected.Render(cursor)
			spacer := m.styles.Selected.Render(" ")
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
			content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ... %d more ...", resultCount-endIdx)) + "\n")
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
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item for batch operations") + "\n")
	content.WriteString(keyStyle.Render("→/l, enter") + descStyle.Render("Open item details") + "\n")
	content.WriteString(keyStyle.Render("i") + descStyle.Render("Insert new item before current") + "\n")
	content.WriteString(keyStyle.Render("a") + descStyle.Render("Append new item after current (or as first child if parent)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete current item or selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of selected items (batch operation)") + "\n")
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

	// Create view keybindings
	content.WriteString(sectionStyle.Render("Create View") + "\n")
	content.WriteString(keyStyle.Render("ctrl+s") + descStyle.Render("Save new item") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Show save/cancel hint") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel creation") + "\n\n")

	// Batch operations
	content.WriteString(sectionStyle.Render("Batch Operations") + "\n")
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item (orange bar shows selection)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete all selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of all selected items") + "\n\n")

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

func (m model) renderCreateView() string {
	var content strings.Builder

	// Title bar
	title := "Create New Work Item"
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

	// Tree edge and icon styles
	edgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// Mode and tab styles (reuse from list view)
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

	// Show loader if creating
	if m.loading {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Creating work item..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()

	// Calculate visible range based on scroll offset
	contentHeight := m.getContentHeight()
	startIdx := m.scrollOffset
	endIdx := m.scrollOffset + contentHeight

	// Total items including the create input line
	totalItems := len(treeItems)
	if m.createInsertPos <= len(treeItems) {
		totalItems++
	}
	if m.hasMoreItems() {
		totalItems++
	}

	// Clamp end index
	if endIdx > totalItems {
		endIdx = totalItems
	}

	// Render visible items with the create input inserted at the correct position
	visibleIdx := 0
	for i := startIdx; i < endIdx; i++ {
		// Check if we should render the create input at this position
		if i == m.createInsertPos {
			// Render the create input line with tree prefix
			var prefix strings.Builder

			// Draw tree prefix based on depth
			for d := 0; d < m.createDepth-1; d++ {
				if d < len(m.createIsLast) && m.createIsLast[d] {
					prefix.WriteString("    ")
				} else {
					prefix.WriteString("│   ")
				}
			}

			// Draw connector for this item
			if m.createDepth > 0 {
				if len(m.createIsLast) > 0 && m.createIsLast[len(m.createIsLast)-1] {
					prefix.WriteString("╰── ")
				} else {
					prefix.WriteString("├── ")
				}
			}

			prefixStr := edgeStyle.Render(prefix.String())

			// Build the create line
			var createLine strings.Builder
			createLine.WriteString(prefixStr)
			createLine.WriteString(iconStyle.Render("✓ "))
			createLine.WriteString(selectedStyle.Render("[New] "))
			createLine.WriteString(m.createInput.View())

			content.WriteString(createLine.String() + "\n")
			visibleIdx++
			continue
		}

		// Calculate actual tree item index (accounting for inserted create line)
		treeIdx := i
		if i > m.createInsertPos {
			treeIdx = i - 1
		}

		// Check if this is a regular tree item
		if treeIdx < len(treeItems) {
			item := treeItems[treeIdx]
			task := item.WorkItem

			// Build tree prefix
			treePrefix := getTreePrefix(item)
			icon := getWorkItemIcon(task.WorkItemType)

			// Get state style
			category := m.getStateCategory(task.State)
			var stateStyle lipgloss.Style
			switch category {
			case "Proposed":
				stateStyle = proposedStateStyle
			case "InProgress":
				stateStyle = inProgressStateStyle
			case "Completed":
				stateStyle = completedStateStyle
			case "Removed":
				stateStyle = removedStateStyle
			default:
				stateStyle = proposedStateStyle
			}

			// Build the line
			var line strings.Builder
			line.WriteString(edgeStyle.Render(treePrefix))
			line.WriteString(iconStyle.Render(icon + " "))
			line.WriteString(stateStyle.Render(fmt.Sprintf("[%s] ", task.State)))
			line.WriteString(stateStyle.Render(fmt.Sprintf("#%d - %s", task.ID, task.Title)))

			content.WriteString(line.String() + "\n")
			visibleIdx++
		} else if treeIdx == len(treeItems) && m.hasMoreItems() {
			// "Load More" item
			remaining := m.getRemainingCount()
			loadMoreText := fmt.Sprintf("▼ Load %d more items...", remaining)
			loadMoreStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Italic(true)
			content.WriteString("  " + loadMoreStyle.Render(loadMoreText) + "\n")
			visibleIdx++
		}
	}

	// Footer with keybindings
	keybindings := "ctrl+s: save • enter: show help • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderErrorView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Error Details"))

	// Error message style
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		MarginBottom(1)

	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Width(m.width - 4).
		PaddingLeft(2)

	// Display the error
	content.WriteString(errorStyle.Render("An error occurred:") + "\n\n")
	content.WriteString(detailStyle.Render(m.statusMessage) + "\n\n")

	// Instructions
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	content.WriteString(helpStyle.Render("This error typically means:") + "\n")
	content.WriteString(helpStyle.Render("1. Missing required permissions in Azure DevOps") + "\n")
	content.WriteString(helpStyle.Render("2. Invalid area path or iteration path") + "\n")
	content.WriteString(helpStyle.Render("3. Work item type not allowed in this project") + "\n\n")

	content.WriteString(helpStyle.Render("Check your Azure DevOps permissions and project settings.") + "\n\n")

	// Footer with keybindings
	keybindings := "esc: back to list • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderDeleteConfirmView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Delete Work Item"))

	// Warning style
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Bold(true)

	questionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	// Display the confirmation message
	content.WriteString(warningStyle.Render("⚠ Warning: This action cannot be undone!") + "\n\n")

	// Check if batch delete or single delete
	if len(m.selectedItems) > 0 {
		// Batch delete - show list of items
		content.WriteString(fmt.Sprintf("Are you sure you want to delete %d work items?\n\n", len(m.selectedItems)))

		// Get current tasks and show titles of selected items
		currentTasks := m.getCurrentTasks()
		taskMap := make(map[int]*WorkItem)
		for i := range currentTasks {
			taskMap[currentTasks[i].ID] = &currentTasks[i]
		}

		// List selected items (limit to 10 for readability)
		listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
		count := 0
		maxDisplay := 10
		for itemID := range m.selectedItems {
			if count >= maxDisplay {
				remaining := len(m.selectedItems) - maxDisplay
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("... and %d more", remaining)) + "\n")
				break
			}
			if task, ok := taskMap[itemID]; ok {
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("• #%d - %s", task.ID, task.Title)) + "\n")
			} else {
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("• #%d", itemID)) + "\n")
			}
			count++
		}

		content.WriteString(questionStyle.Render("\nConfirm batch deletion?") + "\n\n")
	} else {
		// Single delete
		content.WriteString("Are you sure you want to delete this work item?\n\n")
		content.WriteString("  " + itemStyle.Render(fmt.Sprintf("#%d - %s", m.deleteItemID, m.deleteItemTitle)) + "\n")
		content.WriteString(questionStyle.Render("\nConfirm deletion?") + "\n\n")
	}

	content.WriteString("  " + keyStyle.Render("[y]") + " Yes, delete it\n")
	content.WriteString("  " + keyStyle.Render("[n]") + " No, cancel\n\n")

	// Footer with keybindings
	keybindings := "y: confirm delete • n/esc: cancel"
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
