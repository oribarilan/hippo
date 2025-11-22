package main

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// List Management
// ============================================================================

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

// ============================================================================
// Filtering & Navigation
// ============================================================================

// filterSearch performs filtering on the current list based on filter input
func (m *model) filterSearch() {
	query := strings.ToLower(m.filter.filterInput.Value())
	list := m.getCurrentList()
	if list == nil {
		return
	}

	if query == "" {
		list.filteredTasks = nil
		list.filterActive = false
		// Also sync model-level filter state
		m.filter.filteredTasks = nil
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
	m.filter.filteredTasks = filtered
	// Invalidate cache when filter changes
	list.invalidateTreeCache()
}

// getVisibleTasks returns tasks that should be visible based on current filters and mode
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

// hasMoreItems returns true if there are more items to load from the server
func (m model) hasMoreItems() bool {
	if list := m.getCurrentList(); list != nil {
		return list.hasMore()
	}
	return false
}

// getRemainingCount returns the number of items not yet loaded from the server
func (m model) getRemainingCount() int {
	if list := m.getCurrentList(); list != nil {
		return list.getRemainingCount()
	}
	return 0
}

// ============================================================================
// Task Relationships
// ============================================================================

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

// ============================================================================
// UI Calculations
// ============================================================================

// getContentHeight returns the available height for rendering work items
// Takes into account title bar, mode/tab selectors, hint, footer, etc.
func (m model) getContentHeight() int {
	// Calculate fixed UI elements height:
	// - Title bar: 3 lines (title + 2 blank lines)
	// - Mode selector: 2 lines (modes + blank line)
	// - Tab selector: 2 lines (tabs + blank line)
	// - Hint (if present): 2 lines (hint + blank line)
	// - Footer:
	//   * 1 blank line
	//   * 0-1 action log line (if present)
	//   * 1 separator line
	//   * 1 keybindings line
	//   * 2 config bar lines (blank + bar, if config exists)

	fixedHeight := 3 + 2 + 2 // = 7 lines for title, mode, tabs

	// Footer lines
	footerLines := 3 // blank + separator + keybindings (minimum)
	if m.lastActionLog != "" {
		footerLines++ // action log line
	}
	if m.config != nil && m.configSource != nil {
		footerLines += 2 // blank line + config bar
	}
	fixedHeight += footerLines

	// Add hint lines if present
	if m.getTabHint() != "" {
		fixedHeight += 2
	}

	// Add status message lines if present
	if m.statusMessage != "" {
		fixedHeight += 2
	}

	contentHeight := m.ui.height - fixedHeight
	if contentHeight < 5 {
		contentHeight = 5 // Minimum content height
	}

	return contentHeight
}

// adjustScrollOffset adjusts the scroll offset to keep the cursor visible
func (m *model) adjustScrollOffset() {
	contentHeight := m.getContentHeight()

	// Ensure cursor is within visible area
	if m.ui.cursor < m.ui.scrollOffset {
		// Cursor moved above visible area, scroll up
		m.ui.scrollOffset = m.ui.cursor
	} else if m.ui.cursor >= m.ui.scrollOffset+contentHeight {
		// Cursor moved below visible area, scroll down
		m.ui.scrollOffset = m.ui.cursor - contentHeight + 1
	}

	// Ensure scroll offset is never negative
	if m.ui.scrollOffset < 0 {
		m.ui.scrollOffset = 0
	}
}

// ============================================================================
// Display Helpers
// ============================================================================

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

// isWorkItemCompleted returns true if the work item is in the Completed category
func (m model) isWorkItemCompleted(item *WorkItem) bool {
	return m.getStateCategory(item.State) == "Completed"
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

// buildDetailContent creates the content string for the detail view
func (m model) buildDetailContent() string {
	if m.selectedTask == nil {
		return ""
	}

	task := m.selectedTask

	// Use card style with dynamic width
	cardStyle := m.styles.Card.Width(m.ui.width - 4)

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

// ============================================================================
// State Management
// ============================================================================

// setActionLog sets the action log message with the current timestamp
func (m *model) setActionLog(message string) {
	m.lastActionLog = message
	m.lastActionTime = time.Now()
}

// focusEditField focuses the appropriate edit field based on editFieldCursor
func (m *model) focusEditField() {
	// Blur all fields first
	m.edit.titleInput.Blur()
	m.edit.descriptionInput.Blur()

	// Focus the selected field
	switch m.edit.fieldCursor {
	case 0:
		m.edit.titleInput.Focus()
	case 1:
		m.edit.descriptionInput.Focus()
	}
}

// focusWizardField focuses the appropriate wizard field based on fieldCursor
func (m *model) focusWizardField() {
	// Blur all fields first
	m.wizard.orgInput.Blur()
	m.wizard.projectInput.Blur()
	m.wizard.teamInput.Blur()

	// Focus the selected field
	switch m.wizard.fieldCursor {
	case 0:
		m.wizard.orgInput.Focus()
	case 1:
		m.wizard.projectInput.Focus()
	case 2:
		m.wizard.teamInput.Focus()
	}
}
