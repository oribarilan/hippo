package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type viewState int

const (
	loadingView viewState = iota
	listView
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

// UIState contains all UI-related state (cursor, scroll, dimensions)
type UIState struct {
	cursor        int
	scrollOffset  int
	width         int // Terminal width
	height        int // Terminal height
	viewportReady bool
}

// EditState contains all state for edit mode
type EditState struct {
	titleInput       textinput.Model
	descriptionInput textarea.Model
	fieldCursor      int // Which field is currently focused (0=title, 1=description)
	fieldCount       int // Total number of editable fields
}

// CreateState contains all state for create mode
type CreateState struct {
	input         textinput.Model
	insertPos     int    // Position in tree to insert
	after         bool   // true='a', false='i'
	parentID      *int   // nil=parent level, int=child of parent
	depth         int    // Tree depth for rendering
	isLast        []bool // Tree prefix info for rendering
	createdItemID int    // Track newly created item for cursor jump
}

// DeleteState contains all state for delete confirmation
type DeleteState struct {
	itemID    int    // ID of item to delete
	itemTitle string // Title of item to delete (for confirmation message)
}

// BatchState contains state for batch operations
type BatchState struct {
	selectedItems  map[int]bool // Set of selected work item IDs
	operationCount int          // Track pending batch operations
}

// FilterState contains state for filtering and finding
type FilterState struct {
	filteredTasks []WorkItem
	active        bool
	filterInput   textinput.Model
	findInput     textinput.Model
}

type model struct {
	// WorkItemList instances - component-based architecture
	sprintLists  map[sprintTab]*WorkItemList
	backlogLists map[backlogTab]*WorkItemList

	// Core UI state
	state             viewState
	selectedTask      *WorkItem
	selectedTaskID    int // Track which task the viewport is showing
	loading           bool
	loadingMore       bool // Track if we're loading more items (for inline spinner)
	err               error
	client            *AzureDevOpsClient
	spinner           spinner.Model
	viewport          viewport.Model
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

	// Grouped state
	ui     UIState
	edit   EditState
	create CreateState
	delete DeleteState
	batch  BatchState
	filter FilterState

	// UI styles
	styles Styles // Centralized styles for the application
}
