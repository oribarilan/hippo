package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// TEA.MSG HANDLERS
// =============================================================================

// handleTasksLoadedMsg handles the tasksLoadedMsg response
func (m model) handleTasksLoadedMsg(msg tasksLoadedMsg) (model, tea.Cmd) {
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
					m.ui.cursor = list.cursor
					m.ui.scrollOffset = list.scrollOffset
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
					m.ui.cursor = list.cursor
					m.ui.scrollOffset = list.scrollOffset
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

	return m, nil
}

// handleStateUpdatedMsg handles the stateUpdatedMsg response
func (m model) handleStateUpdatedMsg(msg stateUpdatedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		// Show detailed error
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		m.setActionLog(fmt.Sprintf("Error updating state: %v", msg.err))
		m.batch.operationCount = 0 // Reset on error
		m.state = listView
		m.stateCursor = 0
	} else {
		// Success! Decrement counter
		if m.batch.operationCount > 0 {
			m.batch.operationCount--
		}

		// Only refresh when all operations are complete
		if m.batch.operationCount == 0 {
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

	return m, nil
}

// handleSprintUpdatedMsg handles the sprintUpdatedMsg response
func (m model) handleSprintUpdatedMsg(msg sprintUpdatedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		// Show detailed error
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		m.setActionLog(fmt.Sprintf("Error moving to sprint: %v", msg.err))
		m.batch.operationCount = 0 // Reset on error
		m.state = listView
		m.stateCursor = 0
	} else {
		// Success! Decrement counter
		if m.batch.operationCount > 0 {
			m.batch.operationCount--
		}

		// Only refresh when all operations are complete
		if m.batch.operationCount == 0 {
			m.loading = false
			m.statusMessage = ""
			m.state = listView
			m.stateCursor = 0
			m.statusMessage = "Sprint updated successfully!"
			m.setActionLog("Moved items to sprint successfully")

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

	return m, nil
}

// handleWorkItemUpdatedMsg handles the workItemUpdatedMsg response
func (m model) handleWorkItemUpdatedMsg(msg workItemUpdatedMsg) (model, tea.Cmd) {
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

	return m, nil
}

// handleStatesLoadedMsg handles the statesLoadedMsg response
func (m model) handleStatesLoadedMsg(msg statesLoadedMsg) (model, tea.Cmd) {
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

	return m, nil
}

// handleWorkItemRefreshedMsg handles the workItemRefreshedMsg response
func (m model) handleWorkItemRefreshedMsg(msg workItemRefreshedMsg) (model, tea.Cmd) {
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

	return m, nil
}

// handleWorkItemCreatedMsg handles the workItemCreatedMsg response
func (m model) handleWorkItemCreatedMsg(msg workItemCreatedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		// Show detailed error in error view
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		m.setActionLog(fmt.Sprintf("Error creating work item: %v", msg.err))
		// Switch to error view to show full details
		m.state = errorView
	} else if msg.workItem != nil {
		// Success! Store the created item ID and continue with spinner
		m.create.createdItemID = msg.workItem.ID
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

	return m, nil
}

// handleWorkItemDeletedMsg handles the workItemDeletedMsg response
func (m model) handleWorkItemDeletedMsg(msg workItemDeletedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		// Show detailed error
		m.loading = false
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		m.setActionLog(fmt.Sprintf("Error deleting work item: %v", msg.err))
		m.batch.operationCount = 0 // Reset on error
	} else {
		// Success! Decrement counter
		if m.batch.operationCount > 0 {
			m.batch.operationCount--
		}

		// Only refresh when all operations are complete
		if m.batch.operationCount == 0 {
			m.statusMessage = "Refreshing list..."
			m.setActionLog("Deleted work item(s)")

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

	return m, nil
}

// handleSprintsLoadedMsg handles the sprintsLoadedMsg response
func (m model) handleSprintsLoadedMsg(msg sprintsLoadedMsg) (model, tea.Cmd) {
	// Store client first
	if msg.client != nil {
		m.client = msg.client
	}

	if msg.err != nil {
		m.err = fmt.Errorf("failed to load sprints: %w", msg.err)
		m.loading = false
		return m, nil
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
				// Transition from loading screen to list view
				if m.state == loadingView {
					m.state = listView
				}
				loadCmds = append(loadCmds, m.spinner.Tick)
				return m, tea.Batch(loadCmds...)
			}
		}
	}

	// Transition from loading screen to list view if no sprints to load
	if m.state == loadingView {
		m.state = listView
		m.loading = false
	}

	return m, nil
}
