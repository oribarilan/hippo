package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// VIEW-SPECIFIC KEY INPUT HANDLERS
// =============================================================================

// handleHelpView handles keyboard input in the help view
func (m model) handleHelpView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		m.state = listView
		return m, nil
	}
	return m, nil
}

// handleErrorView handles keyboard input in the error view
func (m model) handleErrorView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
		m.statusMessage = ""
		return m, nil
	}
	return m, nil
}

// handleFilterView handles keyboard input in the filter view
func (m model) handleFilterView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
		m.filter.active = false
		m.filter.filterInput.SetValue("")
		// Clear filter in current list
		if list := m.getCurrentList(); list != nil {
			list.filterActive = false
			list.filteredTasks = nil
		}
		m.filter.filteredTasks = nil
		m.ui.cursor = 0
		return m, nil
	case "enter":
		// If there are results, open the selected item in detail view
		if len(m.filter.filteredTasks) > 0 && m.ui.cursor < len(m.filter.filteredTasks) {
			// Get the filtered tasks as tree items to respect the cursor position
			visibleTasks := m.getVisibleTasks()
			if len(visibleTasks) > 0 && m.ui.cursor < len(visibleTasks) {
				treeItems := m.getVisibleTreeItems()
				if m.ui.cursor < len(treeItems) {
					m.selectedTask = treeItems[m.ui.cursor].WorkItem
					m.selectedTaskID = m.selectedTask.ID
					m.state = detailView
					m.filter.active = true
				}
			}
		} else {
			// Just close filter and keep filter active
			m.state = listView
			m.filter.active = true
			m.ui.cursor = 0
		}
		return m, nil
	case "up", "ctrl+k", "ctrl+p":
		// Navigate up in filtered results
		if m.ui.cursor > 0 {
			m.ui.cursor--
			m.adjustScrollOffset()
		}
		return m, nil
	case "down", "ctrl+j", "ctrl+n":
		// Navigate down in filtered results
		treeItems := m.getVisibleTreeItems()
		if m.ui.cursor < len(treeItems)-1 {
			m.ui.cursor++
			m.adjustScrollOffset()
		}
		return m, nil
	case "ctrl+u", "pgup":
		// Jump up half page
		m.ui.cursor = max(0, m.ui.cursor-10)
		m.adjustScrollOffset()
		return m, nil
	case "ctrl+d", "pgdown":
		// Jump down half page
		treeItems := m.getVisibleTreeItems()
		m.ui.cursor = min(len(treeItems)-1, m.ui.cursor+10)
		m.adjustScrollOffset()
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter.filterInput, cmd = m.filter.filterInput.Update(msg)
		m.filterSearch()
		// Reset cursor when filter changes
		m.ui.cursor = 0
		return m, cmd
	}
}

// handleFindView handles keyboard input in the find view
func (m model) handleFindView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
		m.filter.findInput.SetValue("")
		return m, nil
	case "enter":
		// For now, just go back to list view
		// Could implement custom queries here
		m.state = listView
		m.filter.findInput.SetValue("")
		if m.client != nil {
			m.loading = true
			return m, tea.Batch(loadTasks(m.client), loadSprints(m.client), m.spinner.Tick)
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter.findInput, cmd = m.filter.findInput.Update(msg)
		return m, cmd
	}
}

// handleStatePickerView handles keyboard input in the state picker view
func (m model) handleStatePickerView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to appropriate view
		if len(m.batch.selectedItems) > 0 {
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
		if len(m.batch.selectedItems) > 0 && m.client != nil {
			// Batch state update
			m.loading = true
			count := len(m.batch.selectedItems)
			m.batch.operationCount = count // Track batch operations
			m.statusMessage = fmt.Sprintf("Updating %d items to %s...", count, newState)
			m.state = listView

			var updateCmds []tea.Cmd
			for itemID := range m.batch.selectedItems {
				updateCmds = append(updateCmds, updateWorkItemState(m.client, itemID, newState))
			}

			// Clear selection after starting update
			m.batch.selectedItems = make(map[int]bool)
			updateCmds = append(updateCmds, m.spinner.Tick)
			return m, tea.Batch(updateCmds...)
		} else if m.selectedTask != nil && m.client != nil {
			// Single item state update
			m.loading = true
			m.batch.operationCount = 1 // Single operation
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

// handleEditView handles keyboard input in the edit view
func (m model) handleEditView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel edit and return to detail view
		m.state = detailView
		return m, nil
	case "tab":
		// Move to next field
		m.edit.fieldCursor = (m.edit.fieldCursor + 1) % m.edit.fieldCount
		m.focusEditField()
		return m, nil
	case "shift+tab":
		// Move to previous field
		m.edit.fieldCursor--
		if m.edit.fieldCursor < 0 {
			m.edit.fieldCursor = m.edit.fieldCount - 1
		}
		m.focusEditField()
		return m, nil
	case "ctrl+s":
		// Save changes
		if m.selectedTask != nil && m.client != nil {
			updates := make(map[string]interface{})

			// Collect values from inputs
			if title := m.edit.titleInput.Value(); title != "" && title != m.selectedTask.Title {
				updates["title"] = title
			}
			if desc := m.edit.descriptionInput.Value(); desc != m.selectedTask.Description {
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
		switch m.edit.fieldCursor {
		case 0:
			m.edit.titleInput, cmd = m.edit.titleInput.Update(msg)
		case 1:
			m.edit.descriptionInput, cmd = m.edit.descriptionInput.Update(msg)
		}
		return m, cmd
	}
}

// handleCreateView handles keyboard input in the create view
func (m model) handleCreateView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel creation and return to list view
		m.state = listView
		m.create.input.SetValue("")
		return m, nil
	case "enter":
		// Show help hint instead of submitting
		m.statusMessage = "Use ctrl+s to save, esc to cancel"
		return m, nil
	case "ctrl+s":
		// Save new work item
		title := strings.TrimSpace(m.create.input.Value())
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

			if m.create.parentID != nil {
				// Find parent in current tasks to get its area path
				for _, task := range currentTasks {
					if task.ID == *m.create.parentID {
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
				createWorkItem(m.client, title, "Task", iterationPath, m.create.parentID, areaPath),
				m.spinner.Tick,
			)
		}
		return m, nil
	default:
		// Update the input field
		var cmd tea.Cmd
		m.create.input, cmd = m.create.input.Update(msg)
		return m, cmd
	}
}

// handleDeleteConfirmView handles keyboard input in the delete confirmation view
func (m model) handleDeleteConfirmView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		if m.client != nil {
			m.loading = true
			m.state = listView

			// Check if batch delete or single delete
			if len(m.batch.selectedItems) > 0 {
				// Batch delete
				var deleteCmds []tea.Cmd
				count := len(m.batch.selectedItems)
				m.batch.operationCount = count // Track batch operations
				m.statusMessage = fmt.Sprintf("Deleting %d work items...", count)

				for itemID := range m.batch.selectedItems {
					deleteCmds = append(deleteCmds, deleteWorkItem(m.client, itemID))
				}

				// Clear selection after starting delete
				m.batch.selectedItems = make(map[int]bool)
				deleteCmds = append(deleteCmds, m.spinner.Tick)
				return m, tea.Batch(deleteCmds...)
			} else {
				// Single delete
				m.batch.operationCount = 1 // Single operation
				m.statusMessage = "Deleting work item..."
				return m, tea.Batch(deleteWorkItem(m.client, m.delete.itemID), m.spinner.Tick)
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

// handleListViewNav handles navigation in the list view
func (m model) handleListViewNav(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "right", "l":
		// Drill down to detail view
		treeItems := m.getVisibleTreeItems()
		if len(treeItems) > 0 && m.ui.cursor < len(treeItems) {
			m.selectedTask = treeItems[m.ui.cursor].WorkItem
			m.selectedTaskID = m.selectedTask.ID
			m.state = detailView
		}
	case "tab":
		// Cycle through tabs based on current mode
		// Clear selections when switching tabs
		m.batch.selectedItems = make(map[int]bool)

		if m.currentMode == sprintMode {
			m.currentTab = (m.currentTab + 1) % 3
			// Restore cursor/scroll from the new tab's list
			if list := m.getCurrentList(); list != nil {
				m.ui.cursor = list.cursor
				m.ui.scrollOffset = list.scrollOffset
			} else {
				m.ui.cursor = 0
				m.ui.scrollOffset = 0
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
				m.ui.cursor = list.cursor
				m.ui.scrollOffset = list.scrollOffset
			} else {
				m.ui.cursor = 0
				m.ui.scrollOffset = 0
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
		if m.ui.cursor > 0 {
			m.ui.cursor--
			m.adjustScrollOffset()
			// Also update current list's cursor
			if list := m.getCurrentList(); list != nil {
				list.cursor = m.ui.cursor
				list.scrollOffset = m.ui.scrollOffset
			}
		}
	case "down", "j":
		treeItems := m.getVisibleTreeItems()
		maxCursor := len(treeItems) - 1
		// If there are more items to load, allow cursor to go to the "Load More" item
		if m.hasMoreItems() {
			maxCursor = len(treeItems)
		}
		if m.ui.cursor < maxCursor {
			m.ui.cursor++
			m.adjustScrollOffset()
			// Also update current list's cursor
			if list := m.getCurrentList(); list != nil {
				list.cursor = m.ui.cursor
				list.scrollOffset = m.ui.scrollOffset
			}
		}
	case "ctrl+u", "pgup":
		// Jump up half page
		m.ui.cursor = max(0, m.ui.cursor-10)
		m.adjustScrollOffset()
		// Also update current list's cursor
		if list := m.getCurrentList(); list != nil {
			list.cursor = m.ui.cursor
			list.scrollOffset = m.ui.scrollOffset
		}
	case "ctrl+d", "pgdown":
		// Jump down half page
		treeItems := m.getVisibleTreeItems()
		maxCursor := len(treeItems) - 1
		if m.hasMoreItems() {
			maxCursor = len(treeItems)
		}
		m.ui.cursor = min(maxCursor, m.ui.cursor+10)
		m.adjustScrollOffset()
		// Also update current list's cursor
		if list := m.getCurrentList(); list != nil {
			list.cursor = m.ui.cursor
			list.scrollOffset = m.ui.scrollOffset
		}
	case " ":
		// Toggle selection for current item
		treeItems := m.getVisibleTreeItems()
		if len(treeItems) > 0 && m.ui.cursor < len(treeItems) {
			itemID := treeItems[m.ui.cursor].WorkItem.ID
			if m.batch.selectedItems[itemID] {
				delete(m.batch.selectedItems, itemID)
			} else {
				m.batch.selectedItems[itemID] = true
			}
		}
	case "enter":
		treeItems := m.getVisibleTreeItems()
		// Check if cursor is on "Load More" item
		if m.ui.cursor == len(treeItems) && m.hasMoreItems() {
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
		} else if len(treeItems) > 0 && m.ui.cursor < len(treeItems) {
			m.selectedTask = treeItems[m.ui.cursor].WorkItem
			m.selectedTaskID = m.selectedTask.ID
			m.state = detailView
		}
	}
	return m, nil
}

// handleDetailViewNav handles navigation in the detail view
func (m model) handleDetailViewNav(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "left", "h":
		m.state = listView
	}
	return m, nil
}
