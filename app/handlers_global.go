package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// GLOBAL HOTKEYS
// =============================================================================

// handleGlobalHotkeys handles global keyboard shortcuts that work across views
func (m model) handleGlobalHotkeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true

	case "1":
		// Switch to Sprint Mode
		if m.state == listView && m.currentMode != sprintMode {
			m.currentMode = sprintMode
			m.ui.cursor = 0
			m.ui.scrollOffset = 0
			m.setActionLog("Switched to Sprint Mode")
		}
		return m, nil, true

	case "2":
		// Switch to Backlog Mode
		if m.state == listView && m.currentMode != backlogMode {
			m.currentMode = backlogMode
			m.ui.cursor = 0
			m.ui.scrollOffset = 0
			// Load backlog data if not attempted yet
			currentList := m.getCurrentList()
			if currentList != nil && !currentList.attempted && m.client != nil {
				m.loading = true
				tab := m.currentBacklogTab
				return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick), true
			}
			m.setActionLog("Switched to Backlog Mode")
		}
		return m, nil, true

	case "?":
		// Show help modal
		if m.state == helpView {
			m.state = listView
		} else {
			m.state = helpView
		}
		return m, nil, true

	case "r":
		// Refresh
		if m.client != nil {
			// If we're in detail view, just refresh the selected item
			if m.state == detailView && m.selectedTask != nil {
				m.loading = true
				m.statusMessage = "Refreshing item..."
				m.setActionLog(fmt.Sprintf("Refreshing #%d...", m.selectedTask.ID))
				return m, tea.Batch(refreshWorkItem(m.client, m.selectedTask.ID), m.spinner.Tick), true
			}
			// Otherwise, refresh everything based on current mode
			m.loading = true
			m.statusMessage = "Refreshing..."
			m.setActionLog("Refreshing data...")
			m.filter.active = false
			m.filter.filterInput.SetValue("")
			m.filter.filteredTasks = nil

			if m.currentMode == sprintMode {
				// Clear sprint data
				m.sprintLists = make(map[sprintTab]*WorkItemList)
				return m, tea.Batch(loadSprintsWithReload(m.client, true), m.spinner.Tick), true
			} else {
				// Clear backlog data
				m.backlogLists = make(map[backlogTab]*WorkItemList)
				// Reload current backlog tab
				tab := m.currentBacklogTab
				return m, tea.Batch(loadTasksForBacklogTab(m.client, tab, m.getCurrentSprintPath()), m.spinner.Tick), true
			}
		}
		return m, nil, true

	case "o":
		// Open in browser
		var workItemID int
		if m.state == detailView && m.selectedTask != nil {
			workItemID = m.selectedTask.ID
		} else if m.state == listView {
			treeItems := m.getVisibleTreeItems()
			if len(treeItems) > 0 && m.ui.cursor < len(treeItems) {
				workItemID = treeItems[m.ui.cursor].WorkItem.ID
			}
		}
		if workItemID > 0 {
			openInBrowser(m.organizationURL, m.projectName, workItemID)
			m.setActionLog(fmt.Sprintf("Opened #%d in browser", workItemID))
		}
		return m, nil, true

	case "s":
		// Change state - in detail view or list view (for batch)
		if m.state == detailView && m.selectedTask != nil && m.client != nil {
			// Single item state change from detail view
			m.loading = true
			m.statusMessage = "Loading states..."
			return m, tea.Batch(
				loadWorkItemStates(m.client, m.selectedTask.WorkItemType),
				m.spinner.Tick,
			), true
		} else if m.state == listView && len(m.batch.selectedItems) > 0 && m.client != nil {
			// Batch state change from list view
			m.loading = true
			m.statusMessage = "Loading states..."
			// Use "Task" as default work item type for batch operations
			return m, tea.Batch(
				loadWorkItemStates(m.client, "Task"),
				m.spinner.Tick,
			), true
		}
		return m, nil, true

	case "e":
		// Enter edit mode - only in detail view
		if m.state == detailView && m.selectedTask != nil {
			// Populate edit fields with current values
			m.edit.titleInput.SetValue(m.selectedTask.Title)
			m.edit.descriptionInput.SetValue(m.selectedTask.Description)
			m.edit.fieldCursor = 0
			m.edit.titleInput.Focus()
			m.edit.descriptionInput.Blur()
			m.state = editView
		}
		return m, nil, true

	case "/":
		// Filter within existing results
		if m.state == listView {
			m.state = filterView
			m.filter.filterInput.Focus()
			m.filter.active = true // Activate filter immediately
			m.ui.cursor = 0
			m.ui.scrollOffset = 0
		}
		return m, nil, true

	case "f":
		// Find with dedicated query
		if m.state == listView {
			m.state = findView
			m.filter.findInput.Focus()
		}
		return m, nil, true

	case "i", "a":
		// Insert (i) or append (a) new work item - only in list view
		if m.state != listView {
			return m, nil, true
		}

		treeItems := m.getVisibleTreeItems()
		if len(treeItems) == 0 {
			// Empty list - create first item at root level
			m.create.insertPos = 0
			m.create.after = false
			m.create.parentID = nil
			m.create.depth = 0
			m.create.isLast = []bool{}
		} else if m.ui.cursor >= len(treeItems) {
			// On "Load More" item - do nothing
			return m, nil, true
		} else {
			item := treeItems[m.ui.cursor]
			m.create.after = (msg.String() == "a")

			if m.create.after {
				// Append after - check if current item has children or can have children
				// If current item is a parent (has children or could be one), create as first child
				if len(item.WorkItem.Children) > 0 {
					// Has children - insert as first child (right after parent)
					m.create.insertPos = m.ui.cursor + 1
					m.create.depth = item.Depth + 1
					m.create.parentID = &item.WorkItem.ID
					// Append false to IsLast (not last in parent's IsLast chain, plus new child level)
					m.create.isLast = append([]bool{}, item.IsLast...)
					m.create.isLast = append(m.create.isLast, false) // First child, not last
				} else {
					// No children - create as sibling after the subtree
					m.create.insertPos = getPositionAfterSubtree(treeItems, m.ui.cursor)
					// Determine parent and depth based on position after subtree
					if m.create.insertPos < len(treeItems) {
						// There's an item after the subtree - use same depth as cursor item (sibling)
						m.create.depth = item.Depth
						m.create.parentID = getParentIDForTreeItem(item)
						m.create.isLast = append([]bool{}, item.IsLast...)
					} else {
						// Appending at end of list - same depth as cursor item
						m.create.depth = item.Depth
						m.create.parentID = getParentIDForTreeItem(item)
						m.create.isLast = append([]bool{}, item.IsLast...)
						if len(m.create.isLast) > 0 {
							m.create.isLast[len(m.create.isLast)-1] = true // Mark as last
						}
					}
				}
			} else {
				// Insert before - same depth and parent as current item
				m.create.insertPos = m.ui.cursor
				m.create.depth = item.Depth
				m.create.parentID = getParentIDForTreeItem(item)
				m.create.isLast = append([]bool{}, item.IsLast...)
			}
		}

		m.create.input.SetValue("")
		m.create.input.Focus()
		m.state = createView
		return m, nil, true

	case "d":
		// Delete work item(s) - only in list view
		if m.state != listView {
			return m, nil, true
		}

		// Check if we have batch selection
		if len(m.batch.selectedItems) > 0 {
			// Batch delete - just set flag, will be handled in confirmation
			m.state = deleteConfirmView
			return m, nil, true
		}

		// Single delete
		treeItems := m.getVisibleTreeItems()
		if len(treeItems) == 0 || m.ui.cursor >= len(treeItems) {
			// Empty list or on "Load More" - do nothing
			return m, nil, true
		}

		// Store the item to delete and show confirmation
		item := treeItems[m.ui.cursor].WorkItem
		m.delete.itemID = item.ID
		m.delete.itemTitle = item.Title
		m.state = deleteConfirmView
		return m, nil, true
	}

	// No global hotkey was handled
	return m, nil, false
}
