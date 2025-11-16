package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

const version = "v0.1.0"

// defaultLoadLimit is the number of items to load per request
const defaultLoadLimit = 40

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
