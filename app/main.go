package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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
	searchView
	filterView
)

type sprintTab int

const (
	previousSprint sprintTab = iota
	currentSprint
	nextSprint
)

type Sprint struct {
	Name      string
	Path      string
	StartDate string
	EndDate   string
}

type model struct {
	tasks           []WorkItem
	filteredTasks   []WorkItem
	cursor          int
	state           viewState
	selectedTask    *WorkItem
	selectedTaskID  int // Track which task the viewport is showing
	loading         bool
	err             error
	client          *AzureDevOpsClient
	spinner         spinner.Model
	searchInput     textinput.Model
	filterInput     textinput.Model
	viewport        viewport.Model
	viewportReady   bool
	stateCursor     int
	availableStates []string
	stateCategories map[string]string // Map of state name to category (Proposed, InProgress, Completed, etc.)
	organizationURL string
	projectName     string
	searchActive    bool
	statusMessage   string
	lastActionLog   string    // Log line showing the result of the last action
	lastActionTime  time.Time // Timestamp of the last action
	currentTab      sprintTab
	sprints         map[sprintTab]*Sprint
	sprintCounts    map[sprintTab]int  // Total count per sprint
	sprintLoaded    map[sprintTab]int  // Loaded count per sprint
	sprintAttempted map[sprintTab]bool // Whether we've attempted to load this sprint
	initialLoading  int                // Count of initial sprint loads pending
	width           int                // Terminal width
	height          int                // Terminal height
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

	searchInput := textinput.New()
	searchInput.Placeholder = "Search by title or ID..."
	searchInput.Focus()

	filterInput := textinput.New()
	filterInput.Placeholder = "Enter filter (e.g., assigned to me, all items)..."
	filterInput.Focus()

	return model{
		tasks:           []WorkItem{},
		filteredTasks:   []WorkItem{},
		cursor:          0,
		state:           listView,
		loading:         true,
		spinner:         s,
		searchInput:     searchInput,
		filterInput:     filterInput,
		availableStates: []string{"New", "Active", "Closed", "Removed"},
		stateCategories: make(map[string]string),
		organizationURL: os.Getenv("AZURE_DEVOPS_ORG_URL"),
		projectName:     os.Getenv("AZURE_DEVOPS_PROJECT"),
		currentTab:      currentSprint,
		sprints:         make(map[sprintTab]*Sprint),
		sprintCounts:    make(map[sprintTab]int),
		sprintLoaded:    make(map[sprintTab]int),
		sprintAttempted: make(map[sprintTab]bool),
	}
}

type tasksLoadedMsg struct {
	tasks      []WorkItem
	err        error
	client     *AzureDevOpsClient
	totalCount int
	append     bool
	forTab     *sprintTab // Which tab these tasks are for
}

type stateUpdatedMsg struct {
	err error
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
		// Handle search input
		if m.state == searchView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.searchActive = false
				m.searchInput.SetValue("")
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
							m.searchActive = true
						}
					}
				} else {
					// Just close search and keep filter active
					m.state = listView
					m.searchActive = true
					m.cursor = 0
				}
				return m, nil
			case "up", "ctrl+k", "ctrl+p":
				// Navigate up in filtered results
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "ctrl+j", "ctrl+n":
				// Navigate down in filtered results
				treeItems := m.getVisibleTreeItems()
				if m.cursor < len(treeItems)-1 {
					m.cursor++
				}
				return m, nil
			case "ctrl+u":
				// Jump up half page
				m.cursor = max(0, m.cursor-10)
				return m, nil
			case "ctrl+d":
				// Jump down half page
				treeItems := m.getVisibleTreeItems()
				m.cursor = min(len(treeItems)-1, m.cursor+10)
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.filterSearch()
				// Reset cursor when search changes
				m.cursor = 0
				return m, cmd
			}
		}

		// Handle filter input
		if m.state == filterView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.filterInput.SetValue("")
				return m, nil
			case "enter":
				// For now, just go back to list view
				// Could implement custom queries here
				m.state = listView
				m.filterInput.SetValue("")
				if m.client != nil {
					m.loading = true
					return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
				}
				return m, nil
			default:
				m.filterInput, cmd = m.filterInput.Update(msg)
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

		// Global hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r":
			// Refresh
			if m.client != nil {
				m.loading = true
				m.statusMessage = "Refreshing..."
				m.setActionLog("Refreshing data...")
				m.searchActive = false
				m.searchInput.SetValue("")
				m.filteredTasks = nil
				return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
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

		case "/":
			// Search within results
			if m.state == listView {
				m.state = searchView
				m.searchInput.Focus()
				m.searchActive = true // Activate search immediately
				m.cursor = 0
			}
			return m, nil

		case "f":
			// New filter/query
			if m.state == listView {
				m.state = filterView
				m.filterInput.Focus()
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
				// Cycle through tabs
				m.currentTab = (m.currentTab + 1) % 3
				m.cursor = 0
				// Load sprint data if not attempted yet
				if !m.sprintAttempted[m.currentTab] && m.sprints[m.currentTab] != nil && m.client != nil {
					sprint := m.sprints[m.currentTab]
					m.loading = true
					tab := m.currentTab
					return m, tea.Batch(loadTasksForSprint(m.client, nil, sprint.Path, 10, &tab), m.spinner.Tick)
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
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
				}
			case "enter":
				treeItems := m.getVisibleTreeItems()
				// Check if cursor is on "Load More" item
				if m.cursor == len(treeItems) && m.hasMoreItems() {
					// Load more items by excluding already loaded IDs for current sprint
					if m.client != nil {
						m.loading = true
						m.statusMessage = "Loading more items..."

						// Get current sprint path
						sprint := m.sprints[m.currentTab]
						sprintPath := ""
						if sprint != nil {
							sprintPath = sprint.Path
						}

						// Collect all currently loaded IDs in this sprint to exclude
						excludeIDs := make([]int, 0)
						for _, task := range m.tasks {
							if task.IterationPath == sprintPath {
								excludeIDs = append(excludeIDs, task.ID)
							}
						}

						tab := m.currentTab
						return m, tea.Batch(loadTasksForSprint(m.client, excludeIDs, sprintPath, 30, &tab), m.spinner.Tick)
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
			// Determine which tab these tasks are for
			targetTab := m.currentTab
			if msg.forTab != nil {
				targetTab = *msg.forTab
			}

			if msg.append {
				// Append new tasks to existing ones (load more scenario)
				m.tasks = append(m.tasks, msg.tasks...)

				// Update sprint-specific counts
				// Add the new items count to the existing loaded count
				m.sprintLoaded[targetTab] = m.sprintLoaded[targetTab] + len(msg.tasks)
				m.sprintCounts[targetTab] = msg.totalCount

				m.statusMessage = fmt.Sprintf("Loaded %d more items (%d of %d in this sprint)",
					len(msg.tasks), m.sprintLoaded[targetTab], msg.totalCount)
				m.setActionLog(fmt.Sprintf("Loaded %d more items", len(msg.tasks)))
				m.loading = false
			} else {
				// Initial load or replace
				if len(m.tasks) == 0 {
					// Very first load
					m.tasks = msg.tasks
				} else {
					// Append to existing (for other sprint loads)
					m.tasks = append(m.tasks, msg.tasks...)
				}
				m.filteredTasks = nil
				if targetTab == m.currentTab {
					m.cursor = 0
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
		if m.loading {
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
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredTasks = nil
		return
	}

	var filtered []WorkItem
	for _, task := range m.tasks {
		if strings.Contains(strings.ToLower(task.Title), query) ||
			strings.Contains(fmt.Sprintf("%d", task.ID), query) {
			filtered = append(filtered, task)
		}
	}
	m.filteredTasks = filtered
}

func (m model) getVisibleTasks() []WorkItem {
	tasks := m.tasks

	// Apply search filter if active
	if m.searchActive && m.filteredTasks != nil {
		tasks = m.filteredTasks
	}

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
	// Check if there are more items in the current sprint that we haven't loaded yet
	loaded := m.sprintLoaded[m.currentTab]
	total := m.sprintCounts[m.currentTab]
	return loaded < total
}

func (m model) getRemainingCount() int {
	loaded := m.sprintLoaded[m.currentTab]
	total := m.sprintCounts[m.currentTab]
	return total - loaded
}

// getParentTask finds and returns the parent task if it exists
func (m model) getParentTask(task *WorkItem) *WorkItem {
	if task == nil || task.ParentID == nil {
		return nil
	}

	// Search through all tasks to find the parent
	for i := range m.tasks {
		if m.tasks[i].ID == *task.ParentID {
			return &m.tasks[i]
		}
	}

	return nil
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
		relativeTime := getRelativeTime(task.CreatedDate)
		dateValue := task.CreatedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", task.CreatedDate, relativeTime)
		}
		cardContent.WriteString(labelStyle.Render("Created:"))
		cardContent.WriteString(valueStyle.Render(dateValue))
		cardContent.WriteString("\n")
	}

	if task.ChangedDate != "" {
		relativeTime := getRelativeTime(task.ChangedDate)
		dateValue := task.ChangedDate
		if relativeTime != "" {
			dateValue = fmt.Sprintf("%s %s", task.ChangedDate, relativeTime)
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

func (m model) getVisibleTasksCount() int {
	// Get count of visible tasks in current sprint
	sprint := m.sprints[m.currentTab]
	if sprint == nil || sprint.Path == "" {
		return len(m.tasks)
	}

	count := 0
	for _, task := range m.tasks {
		if task.IterationPath == sprint.Path {
			count++
		}
	}
	return count
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
	if m.loading {
		return fmt.Sprintf("\n  %s %s\n\n", m.spinner.View(), m.statusMessage)
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n", m.err)
	}

	switch m.state {
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
	case searchView:
		return m.renderSearchView()
	case filterView:
		return m.renderFilterView()
	default:
		return m.renderListView()
	}
}

func (m model) renderListView() string {
	var content strings.Builder

	// Title bar
	title := "Azure DevOps - Work Items"
	if m.searchActive {
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

	// Tab styles
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	// Render tabs
	tabs := []string{}

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

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	if m.statusMessage != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		content.WriteString(msgStyle.Render(m.statusMessage) + "\n\n")
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

	for i, treeItem := range treeItems {
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

	// Add "Load More" or "Load All" item if there are more items
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

		content.WriteString(loadMoreText + "\n")
	}

	// Footer with keybindings
	keybindings := "tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate\nenter: details • o: open in browser • /: search • f: filter • r: refresh • q: quit"
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
	keybindings := "←/h/esc: back • o: open in browser • s: change state • q: quit"
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
	keybindings := "↑/↓ or j/k: navigate • enter: select • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderSearchView() string {
	var content strings.Builder

	// Title bar
	titleText := "Search"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.searchInput.View() + "\n\n")

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
	if m.searchInput.Value() != "" {
		content.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d", resultCount, len(m.tasks))) + "\n\n")
	} else {
		content.WriteString(dimStyle.Render(fmt.Sprintf("  %d items", len(m.tasks))) + "\n\n")
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
	keybindings := "↑/↓ or ctrl+j/k: navigate • ctrl+d/u: half page • enter: open detail • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderFilterView() string {
	var content strings.Builder

	// Title bar
	titleText := "Filter Work Items"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.filterInput.View() + "\n\n")
	content.WriteString("Examples:\n")
	content.WriteString("  - Press Enter to query all items\n")
	content.WriteString("  - (Custom filters coming soon)\n")

	// Footer with keybindings
	keybindings := "enter: apply filter • esc: cancel"
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
