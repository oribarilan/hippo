package main

import (
	"fmt"
	"strings"
)

// renderMoveChildrenConfirmView renders the confirmation dialog for moving children with parents
func (m model) renderMoveChildrenConfirmView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Move Children with Parent?"))

	// Display the confirmation message
	content.WriteString(m.styles.Warning.Render("📦 Parent Item with Children Detected") + "\n\n")

	// Show information about the operation
	childCount := m.sprintMove.childCount
	skippedCount := m.sprintMove.skippedCount

	content.WriteString(fmt.Sprintf("Moving parent item to %s\n", m.sprintMove.targetName))

	if childCount > 0 {
		if skippedCount > 0 {
			content.WriteString(fmt.Sprintf("This parent has %d active child item(s) (%d completed items will be skipped)\n\n", childCount, skippedCount))
		} else {
			content.WriteString(fmt.Sprintf("This parent has %d child item(s)\n\n", childCount))
		}
	}

	// Show hierarchical tree of items to be moved
	if len(m.sprintMove.itemsToMove) > 0 {
		content.WriteString(m.styles.Section.Render("Items to be moved:") + "\n\n")
		m.renderFilteredTreeForConfirm(&content, m.sprintMove.itemsToMove)
		content.WriteString("\n")
	}

	// Show action options
	if childCount > 0 {
		content.WriteString("Would you like to move the children as well?\n\n")
		content.WriteString("  " + m.styles.Key.Render("[y]") + " Yes, move parent and children together\n")
		content.WriteString("  " + m.styles.Key.Render("[n]") + " No, move only the parent item\n")
		content.WriteString("  " + m.styles.Key.Render("[esc]") + " Cancel the operation\n\n")
	} else {
		// This shouldn't happen, but handle it gracefully
		content.WriteString("No children detected. The parent will be moved.\n\n")
		content.WriteString("  " + m.styles.Key.Render("[enter]") + " Continue\n")
		content.WriteString("  " + m.styles.Key.Render("[esc]") + " Cancel\n\n")
	}

	// Footer with keybindings
	keybindings := "y: move all • n: parent only • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

// renderFilteredTreeForConfirm renders the filtered tree structure for the confirmation dialog
func (m model) renderFilteredTreeForConfirm(content *strings.Builder, treeItems []TreeItem) {
	for _, treeItem := range treeItems {
		m.renderTreeItemForConfirm(content, treeItem, 0, []bool{})
	}
}

// renderTreeItemForConfirm recursively renders a tree item and its children with tree prefixes
func (m model) renderTreeItemForConfirm(content *strings.Builder, treeItem TreeItem, depth int, isLast []bool) {
	// Build tree prefix
	var prefix strings.Builder
	for i := 0; i < depth; i++ {
		if i < len(isLast) && isLast[i] {
			prefix.WriteString("    ")
		} else {
			prefix.WriteString("│   ")
		}
	}

	// Add connector
	if depth > 0 {
		if len(isLast) > 0 && isLast[len(isLast)-1] {
			prefix.WriteString("╰── ")
		} else {
			prefix.WriteString("├── ")
		}
	}

	// Format item: prefix + ID + title + state
	item := treeItem.WorkItem
	itemText := fmt.Sprintf("#%d: %s", item.ID, item.Title)
	if len(itemText) > 70 {
		itemText = itemText[:67] + "..."
	}
	stateText := fmt.Sprintf("(%s)", item.State)

	line := fmt.Sprintf("  %s%s %s\n",
		m.styles.TreeEdge.Render(prefix.String()),
		m.styles.Dim.Render(itemText),
		m.styles.Dim.Render(stateText),
	)
	content.WriteString(line)

	// Render children recursively
	for i, child := range item.Children {
		childIsLast := append([]bool{}, isLast...)
		childIsLast = append(childIsLast, i == len(item.Children)-1)

		// Create TreeItem for child
		childTreeItem := TreeItem{
			WorkItem: child,
			Depth:    depth + 1,
			IsLast:   childIsLast,
		}
		m.renderTreeItemForConfirm(content, childTreeItem, depth+1, childIsLast)
	}
}
