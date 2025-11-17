package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleMoveChildrenConfirmView handles keyboard input in the move children confirmation view
func (m model) handleMoveChildrenConfirmView(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel the entire operation - clear batch selection and return to list view
		m.batch.selectedItems = make(map[int]bool)
		m.state = listView
		m.stateCursor = 0
		m.statusMessage = "Sprint move cancelled"
		return m, nil

	case "y", "Y":
		// Yes - move parents AND children
		m.sprintMove.includeChildren = true
		return m.executeSprintMove()

	case "n", "N":
		// No - move only parents
		m.sprintMove.includeChildren = false
		return m.executeSprintMove()

	case "enter":
		// If there are no children, just continue with the move
		if m.sprintMove.childCount == 0 {
			m.sprintMove.includeChildren = false
			return m.executeSprintMove()
		}
	}

	return m, nil
}

// executeSprintMove performs the actual sprint move operation
func (m model) executeSprintMove() (model, tea.Cmd) {
	if len(m.batch.selectedItems) == 0 || m.client == nil {
		m.state = listView
		return m, nil
	}

	m.loading = true

	// Use the filtered tree that was already built during the confirmation stage
	// This ensures we only move non-completed items
	var itemsToMove []int
	if m.sprintMove.includeChildren {
		// Collect all IDs from the filtered tree (parents and children)
		for _, treeItem := range m.sprintMove.itemsToMove {
			itemIDs := m.collectTreeItemIDs(treeItem)
			itemsToMove = append(itemsToMove, itemIDs...)
		}
	} else {
		// Just move the parents (top-level items from filtered tree)
		for _, treeItem := range m.sprintMove.itemsToMove {
			itemsToMove = append(itemsToMove, treeItem.WorkItem.ID)
		}
	}

	count := len(itemsToMove)
	m.batch.operationCount = count

	// Build status message
	if m.sprintMove.includeChildren && m.sprintMove.childCount > 0 {
		if m.sprintMove.skippedCount > 0 {
			m.statusMessage = fmt.Sprintf("Moving %d items (including %d children, skipping %d completed) to %s...",
				count, m.sprintMove.childCount, m.sprintMove.skippedCount, m.sprintMove.targetName)
		} else {
			m.statusMessage = fmt.Sprintf("Moving %d items (including %d children) to %s...",
				count, m.sprintMove.childCount, m.sprintMove.targetName)
		}
	} else {
		if m.sprintMove.skippedCount > 0 {
			m.statusMessage = fmt.Sprintf("Moving %d items (skipping %d completed) to %s...",
				count, m.sprintMove.skippedCount, m.sprintMove.targetName)
		} else {
			m.statusMessage = fmt.Sprintf("Moving %d items to %s...", count, m.sprintMove.targetName)
		}
	}
	m.state = listView

	var updateCmds []tea.Cmd
	for _, itemID := range itemsToMove {
		updateCmds = append(updateCmds, moveWorkItemToSprint(m.client, itemID, m.sprintMove.targetPath))
	}

	// Clear selection after starting update
	m.batch.selectedItems = make(map[int]bool)
	updateCmds = append(updateCmds, m.spinner.Tick)
	return m, tea.Batch(updateCmds...)
}

// collectTreeItemIDs recursively collects all work item IDs from a tree item and its children
func (m model) collectTreeItemIDs(treeItem TreeItem) []int {
	var ids []int
	ids = append(ids, treeItem.WorkItem.ID)

	// Recursively collect child IDs
	for _, child := range treeItem.WorkItem.Children {
		childTreeItem := TreeItem{WorkItem: child}
		childIDs := m.collectTreeItemIDs(childTreeItem)
		ids = append(ids, childIDs...)
	}

	return ids
}
