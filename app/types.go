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
	batchEditMenuView
	sprintPickerView
	moveChildrenConfirmView
	configWizardView
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

// SprintMoveState contains state for sprint move operation
type SprintMoveState struct {
	targetPath      string     // Target sprint path
	targetName      string     // Target sprint name for display
	parentIDs       []int      // Parent work item IDs being moved
	includeChildren bool       // Whether to move children as well
	childCount      int        // Total number of children that would be moved
	itemsToMove     []TreeItem // Tree structure of items that will be moved (for display)
	skippedCount    int        // Number of completed items that were skipped
}

// WizardState contains state for the configuration wizard
type WizardState struct {
	fieldCursor  int // Which field is currently focused (0=org, 1=project, 2=team)
	orgInput     textinput.Model
	projectInput textinput.Model
	teamInput    textinput.Model
	err          string // Validation error message
}

type model struct {
	// Configuration
	config       *Config
	configSource *ConfigSource // Track where config values came from

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
	statusMessage     string
	lastActionLog     string    // Log line showing the result of the last action
	lastActionTime    time.Time // Timestamp of the last action
	currentMode       appMode
	currentTab        sprintTab
	currentBacklogTab backlogTab
	sprints           map[sprintTab]*Sprint
	initialLoading    int // Count of initial sprint loads pending

	// Grouped state
	ui         UIState
	edit       EditState
	create     CreateState
	delete     DeleteState
	batch      BatchState
	filter     FilterState
	sprintMove SprintMoveState
	wizard     WizardState

	// UI styles
	styles Styles // Centralized styles for the application
}
