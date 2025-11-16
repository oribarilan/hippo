package main

import (
	"fmt"
)

// renderLoadMoreItem renders the "Load More" / spinner line for the list view.
// remaining > 30 renders "+30" otherwise renders all remaining.
func (m model) renderLoadMoreItem(isSelected bool, remaining int, loadingMore bool) string {
	cursor := " "
	if isSelected {
		cursor = ">"
	}

	var text string
	if loadingMore {
		text = fmt.Sprintf("%s %s Loading more items...", cursor, m.spinner.View())
	} else {
		if remaining > 30 {
			text = fmt.Sprintf("%s Load More (+30)", cursor)
		} else {
			text = fmt.Sprintf("%s Load All (+%d)", cursor, remaining)
		}
	}

	if isSelected {
		return m.styles.Selected.Render(text)
	}
	return m.styles.LoadMore.Render(text)
}

// renderTreeItemList renders a single work item line for the list view with
// batch selection, cursor, icon, title and state styling.
func (m model) renderTreeItemList(treeItem TreeItem, isSelected bool, isBatchSelected bool) string {
	// Cursor symbol
	cursor := " "
	if isSelected {
		cursor = "❯"
	}

	// Batch indicator bar
	batchIndicator := "  "
	if isBatchSelected {
		batchIndicator = m.styles.BatchIndicator.Render("█ ") // Full block
	}

	// Tree prefix (styled per selection)
	prefixRaw := getTreePrefix(treeItem)
	var treePrefix string
	if isSelected {
		treePrefix = m.styles.TreeEdgeSelected.Render(prefixRaw)
	} else {
		treePrefix = m.styles.TreeEdge.Render(prefixRaw)
	}

	// Icon styling
	iconChar := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
	var iconStyled string
	if isSelected {
		iconStyled = m.styles.IconSelected.Render(iconChar)
	} else {
		iconStyled = m.styles.Icon.Render(iconChar)
	}

	// Category + styles
	category := m.getStateCategory(treeItem.WorkItem.State)
	stateStyle := m.styles.GetStateStyle(category, isSelected)
	titleStyle := m.styles.GetItemTitleStyle(category, isSelected, len(treeItem.WorkItem.Children) > 0)
	stateText := stateStyle.Render(treeItem.WorkItem.State)
	titleText := titleStyle.Render(treeItem.WorkItem.Title)

	if isSelected {
		// Apply background to cursor spacing for visual consistency
		cursorStyled := m.styles.Selected.Render(cursor)
		spacer := m.styles.Selected.Render(" ")
		return fmt.Sprintf("%s%s%s%s%s%s%s%s",
			batchIndicator,
			cursorStyled,
			spacer,
			treePrefix,
			iconStyled,
			spacer,
			titleText,
			spacer+stateText,
		)
	}

	// Non-selected formatting
	return fmt.Sprintf("%s%s %s%s %s %s",
		batchIndicator,
		cursor,
		treePrefix,
		iconStyled,
		titleText,
		stateText,
	)
}

// renderTreeItemFilter renders a tree item line for the filter view (simpler – no batch indicator).
func (m model) renderTreeItemFilter(treeItem TreeItem, isSelected bool) string {
	cursor := "  "
	if isSelected {
		cursor = "❯ "
	}

	prefixRaw := getTreePrefix(treeItem)
	var treePrefix string
	if isSelected {
		treePrefix = m.styles.TreeEdgeSelected.Render(prefixRaw)
	} else {
		treePrefix = m.styles.TreeEdge.Render(prefixRaw)
	}

	iconChar := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
	var iconStyled string
	if isSelected {
		iconStyled = m.styles.IconSelected.Render(iconChar)
	} else {
		iconStyled = m.styles.Icon.Render(iconChar)
	}

	category := m.getStateCategory(treeItem.WorkItem.State)
	stateStyle := m.styles.GetStateStyle(category, isSelected)
	titleStyle := m.styles.GetItemTitleStyle(category, isSelected, len(treeItem.WorkItem.Children) > 0)
	titleText := titleStyle.Render(treeItem.WorkItem.Title)
	stateText := stateStyle.Render(treeItem.WorkItem.State)

	if isSelected {
		cursorStyled := m.styles.Selected.Render(cursor)
		spacer := m.styles.Selected.Render(" ")
		return fmt.Sprintf("%s%s%s%s%s%s",
			cursorStyled,
			treePrefix,
			iconStyled,
			spacer,
			titleText,
			spacer+stateText,
		)
	}

	return fmt.Sprintf("%s%s%s %s %s",
		cursor,
		treePrefix,
		iconStyled,
		titleText,
		stateText,
	)
}

// renderTreeItemCreate renders a tree item line used in create view context (state + id + title)
func (m model) renderTreeItemCreate(treeItem TreeItem) string {
	prefixRaw := getTreePrefix(treeItem)
	prefix := m.styles.TreeEdge.Render(prefixRaw)
	iconChar := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
	iconStyled := m.styles.Icon.Render(iconChar + " ")
	category := m.getStateCategory(treeItem.WorkItem.State)
	stateStyle := m.styles.GetStateStyle(category, false)
	statePart := stateStyle.Render(fmt.Sprintf("[%s] ", treeItem.WorkItem.State))
	titlePart := stateStyle.Render(fmt.Sprintf("#%d - %s", treeItem.WorkItem.ID, treeItem.WorkItem.Title))
	return fmt.Sprintf("%s%s%s%s", prefix, iconStyled, statePart, titlePart)
}
