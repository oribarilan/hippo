package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSprintPickerView handles keyboard input in the sprint picker view
func (m model) handleSprintPickerView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear batch selection and return to list view
		m.batch.selectedItems = make(map[int]bool)
		m.state = listView
		m.stateCursor = 0
		return m, nil

	case "up", "k":
		if m.stateCursor > 0 {
			m.stateCursor--
		}

	case "down", "j":
		// Count available options dynamically
		maxOptions := 0 // Backlog
		if m.sprints[previousSprint] != nil {
			maxOptions++
		}
		if m.sprints[currentSprint] != nil {
			maxOptions++
		}
		if m.sprints[nextSprint] != nil {
			maxOptions++
		}

		if m.stateCursor < maxOptions {
			m.stateCursor++
		}

	case "enter":
		// Build the same options list as in the view
		options := []struct {
			name string
			path string
		}{
			{"Backlog (no sprint)", ""},
		}

		if m.sprints[previousSprint] != nil {
			options = append(options, struct {
				name string
				path string
			}{
				name: fmt.Sprintf("Previous Sprint - %s", m.sprints[previousSprint].Name),
				path: m.sprints[previousSprint].Path,
			})
		}

		if m.sprints[currentSprint] != nil {
			options = append(options, struct {
				name string
				path string
			}{
				name: fmt.Sprintf("Current Sprint - %s", m.sprints[currentSprint].Name),
				path: m.sprints[currentSprint].Path,
			})
		}

		if m.sprints[nextSprint] != nil {
			options = append(options, struct {
				name string
				path string
			}{
				name: fmt.Sprintf("Next Sprint - %s", m.sprints[nextSprint].Name),
				path: m.sprints[nextSprint].Path,
			})
		}

		// Get the selected sprint path
		if m.stateCursor >= 0 && m.stateCursor < len(options) {
			targetPath := options[m.stateCursor].path
			targetName := options[m.stateCursor].name

			if len(m.batch.selectedItems) > 0 && m.client != nil {
				// Check if this is a batch operation (multiple items) or single item
				isBatchMode := len(m.batch.selectedItems) > 1

				if isBatchMode {
					// BATCH MODE: Move only the selected items, no filtering, no confirmation
					m.loading = true
					count := len(m.batch.selectedItems)
					m.batch.operationCount = count
					m.statusMessage = fmt.Sprintf("Moving %d items to %s...", count, targetName)
					m.state = listView

					var updateCmds []tea.Cmd
					for itemID := range m.batch.selectedItems {
						updateCmds = append(updateCmds, moveWorkItemToSprint(m.client, itemID, targetPath))
					}

					// Clear selection after starting update
					m.batch.selectedItems = make(map[int]bool)
					updateCmds = append(updateCmds, m.spinner.Tick)
					return m, tea.Batch(updateCmds...)
				} else {
					// SINGLE ITEM MODE: Check for parent with children, filter completed, show confirmation
					treeItems := m.getVisibleTreeItems()
					taskMap := make(map[int]*WorkItem)
					for i := range treeItems {
						taskMap[treeItems[i].WorkItem.ID] = treeItems[i].WorkItem
					}

					// Get the single selected item
					var selectedItemID int
					for itemID := range m.batch.selectedItems {
						selectedItemID = itemID
						break
					}

					if task, ok := taskMap[selectedItemID]; ok {
						// Check if this is a parent with children
						if len(task.Children) > 0 {
							// Build filtered tree (excluding completed children)
							treeItem := m.findTreeItem(treeItems, selectedItemID)
							if treeItem != nil {
								totalSkippedCount := 0
								filteredTreeItem := m.filterCompletedFromTree(*treeItem, &totalSkippedCount)

								if filteredTreeItem != nil {
									// Count non-completed descendants
									totalChildCount := m.countNonCompletedDescendants(task)

									if totalChildCount > 0 {
										// Show confirmation dialog
										m.sprintMove.targetPath = targetPath
										m.sprintMove.targetName = targetName
										m.sprintMove.parentIDs = []int{selectedItemID}
										m.sprintMove.childCount = totalChildCount
										m.sprintMove.itemsToMove = []TreeItem{*filteredTreeItem}
										m.sprintMove.skippedCount = totalSkippedCount
										m.state = moveChildrenConfirmView
										return m, nil
									}
								}
							}
						}

						// Not a parent or no children - just move the single item
						m.loading = true
						m.batch.operationCount = 1
						m.statusMessage = fmt.Sprintf("Moving item to %s...", targetName)
						m.state = listView

						updateCmd := moveWorkItemToSprint(m.client, selectedItemID, targetPath)
						m.batch.selectedItems = make(map[int]bool)
						return m, tea.Batch(updateCmd, m.spinner.Tick)
					}
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// findTreeItem finds a tree item by work item ID
func (m model) findTreeItem(treeItems []TreeItem, itemID int) *TreeItem {
	for i := range treeItems {
		if treeItems[i].WorkItem.ID == itemID {
			return &treeItems[i]
		}
	}
	return nil
}

// filterCompletedFromTree recursively filters out completed items from a tree
func (m model) filterCompletedFromTree(treeItem TreeItem, skippedCount *int) *TreeItem {
	// If this item is completed, skip it entirely
	if m.isWorkItemCompleted(treeItem.WorkItem) {
		*skippedCount++
		return nil
	}

	// Filter children recursively
	var filteredChildren []*WorkItem
	for _, child := range treeItem.WorkItem.Children {
		if !m.isWorkItemCompleted(child) {
			// Create a filtered copy of the child
			childCopy := *child
			childCopy.Children = m.filterCompletedChildren(child.Children, skippedCount)
			filteredChildren = append(filteredChildren, &childCopy)
		} else {
			*skippedCount++
		}
	}

	// Create a copy with filtered children
	result := treeItem
	workItemCopy := *treeItem.WorkItem
	workItemCopy.Children = filteredChildren
	result.WorkItem = &workItemCopy
	return &result
}

// filterCompletedChildren recursively filters completed children
func (m model) filterCompletedChildren(children []*WorkItem, skippedCount *int) []*WorkItem {
	var filtered []*WorkItem
	for _, child := range children {
		if !m.isWorkItemCompleted(child) {
			childCopy := *child
			childCopy.Children = m.filterCompletedChildren(child.Children, skippedCount)
			filtered = append(filtered, &childCopy)
		} else {
			*skippedCount++
		}
	}
	return filtered
}

// countNonCompletedDescendants recursively counts non-completed descendants of a work item
func (m model) countNonCompletedDescendants(item *WorkItem) int {
	count := 0
	for _, child := range item.Children {
		if !m.isWorkItemCompleted(child) {
			count++
			count += m.countNonCompletedDescendants(child)
		}
	}
	return count
}

// countAllDescendants recursively counts all descendants of a work item
func (m model) countAllDescendants(item *WorkItem) int {
	count := len(item.Children)
	for _, child := range item.Children {
		count += m.countAllDescendants(child)
	}
	return count
}
