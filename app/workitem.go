package main

import (
	"fmt"
	"strings"
	"time"
)

// WorkItem represents a single work item from Azure DevOps
type WorkItem struct {
	ID            int
	Title         string
	State         string
	AssignedTo    string
	WorkItemType  string
	Description   string
	Tags          string
	Priority      int
	CreatedDate   string
	ChangedDate   string
	IterationPath string
	AreaPath      string
	ParentID      *int
	Children      []*WorkItem
	Comments      string // Discussion/History
}

// TreeItem represents a flattened tree view item with depth information
type TreeItem struct {
	WorkItem *WorkItem
	Depth    int
	IsLast   []bool // Track if ancestor at each level is the last child
}

// WorkItemList represents an independent list of work items with its own state
type WorkItemList struct {
	tasks         []WorkItem // The actual work items
	cursor        int        // Current cursor position in this list
	scrollOffset  int        // Scroll offset for this list
	filterActive  bool       // Whether filter is active for this list
	filteredTasks []WorkItem // Filtered tasks for this list
	loaded        int        // Number of items loaded so far
	totalCount    int        // Total count from server
	attempted     bool       // Whether we've attempted to load this list
	// Cache fields for tree structure optimization
	treeCache    []TreeItem // Cached tree structure to avoid rebuilding on every render
	cacheVersion int        // Incremented when tasks change to invalidate cache
}

// Tree Structure Functions

// buildTreeStructure organizes work items into a parent-child hierarchy
func buildTreeStructure(items []WorkItem) []*WorkItem {
	// Create a map of all items by ID for quick lookup
	itemMap := make(map[int]*WorkItem)
	for i := range items {
		itemMap[items[i].ID] = &items[i]
		items[i].Children = nil // Reset children
	}

	// Build parent-child relationships and collect root items
	var roots []*WorkItem
	for i := range items {
		item := &items[i]
		if item.ParentID != nil {
			// This item has a parent
			if parent, exists := itemMap[*item.ParentID]; exists {
				// Parent exists in our list, add as child
				parent.Children = append(parent.Children, item)
			} else {
				// Parent not in our list, treat as root
				roots = append(roots, item)
			}
		} else {
			// No parent, this is a root item
			roots = append(roots, item)
		}
	}

	return roots
}

// flattenTree converts a tree structure into a flat list with depth information
func flattenTree(roots []*WorkItem) []TreeItem {
	var result []TreeItem

	var traverse func(item *WorkItem, depth int, isLast []bool)
	traverse = func(item *WorkItem, depth int, isLast []bool) {
		result = append(result, TreeItem{
			WorkItem: item,
			Depth:    depth,
			IsLast:   append([]bool{}, isLast...), // Copy the slice
		})

		// Recursively add children
		for i, child := range item.Children {
			// Create new isLast slice for this child
			childIsLast := append([]bool{}, isLast...)
			childIsLast = append(childIsLast, i == len(item.Children)-1)
			traverse(child, depth+1, childIsLast)
		}
	}

	for i, root := range roots {
		isLast := []bool{i == len(roots)-1}
		traverse(root, 0, isLast)
	}

	return result
}

// getTreePrefix returns the tree drawing prefix for a tree item with enhanced styling
func getTreePrefix(treeItem TreeItem) string {
	if treeItem.Depth == 0 {
		return ""
	}

	var prefix strings.Builder

	// Draw vertical lines and spaces for each level except the last
	for i := 0; i < treeItem.Depth-1; i++ {
		if treeItem.IsLast[i] {
			prefix.WriteString("    ") // No vertical line if parent was last
		} else {
			prefix.WriteString("│   ") // Vertical line if parent has more siblings
		}
	}

	// Draw the connector for this item with rounded corners
	if len(treeItem.IsLast) > 0 && treeItem.IsLast[len(treeItem.IsLast)-1] {
		prefix.WriteString("╰── ") // Last child with rounded corner
	} else {
		prefix.WriteString("├── ") // Not last child
	}

	return prefix.String()
}

// getWorkItemIcon returns an icon based on the work item type
func getWorkItemIcon(workItemType string) string {
	switch strings.ToLower(workItemType) {
	case "task":
		return "✓"
	default:
		return "•"
	}
}

// countTreeItems counts the total number of items in a tree
func countTreeItems(items []TreeItem) int {
	return len(items)
}

// getPositionAfterSubtree finds the position after an item's entire subtree
func getPositionAfterSubtree(treeItems []TreeItem, pos int) int {
	if pos >= len(treeItems) {
		return pos + 1
	}
	currentDepth := treeItems[pos].Depth
	// Find next item at same or lower depth
	for i := pos + 1; i < len(treeItems); i++ {
		if treeItems[i].Depth <= currentDepth {
			return i
		}
	}
	return len(treeItems) // Append at end
}

// getParentIDForTreeItem returns the parent ID for creating a sibling of the given tree item
func getParentIDForTreeItem(item TreeItem) *int {
	return item.WorkItem.ParentID
}

// WorkItemList Methods

// hasMore returns true if there are more items to load from the server
func (wl *WorkItemList) hasMore() bool {
	return wl.loaded < wl.totalCount
}

// getRemainingCount returns the number of items not yet loaded
func (wl *WorkItemList) getRemainingCount() int {
	return wl.totalCount - wl.loaded
}

// getVisibleTasks returns the appropriate task list (filtered or all)
func (wl *WorkItemList) getVisibleTasks() []WorkItem {
	if wl.filterActive && wl.filteredTasks != nil {
		return wl.filteredTasks
	}
	return wl.tasks
}

// invalidateTreeCache invalidates the cached tree structure
func (wl *WorkItemList) invalidateTreeCache() {
	wl.cacheVersion++
	wl.treeCache = nil
}

// appendTasks adds new tasks to the list and updates loaded count
func (wl *WorkItemList) appendTasks(tasks []WorkItem) {
	wl.tasks = append(wl.tasks, tasks...)
	wl.loaded = len(wl.tasks)
	wl.invalidateTreeCache()
}

// replaceTasks replaces all tasks with new ones and updates counts
func (wl *WorkItemList) replaceTasks(tasks []WorkItem, totalCount int) {
	wl.tasks = tasks
	wl.loaded = len(tasks)
	wl.totalCount = totalCount
	wl.attempted = true
	wl.invalidateTreeCache()
}

// Date/Time Formatting Functions

// formatDateTime formats a datetime string into a human-readable format
func formatDateTime(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Parse the date string (format: 2006-01-02T15:04:05)
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// Try without the time part
		t, err = time.Parse("2006-01-02", dateStr[:10])
		if err != nil {
			return dateStr
		}
	}

	// Format as "Jan 2, 2006 3:04 PM"
	return t.Format("Jan 2, 2006 3:04 PM")
}

// getRelativeTime returns a human-readable relative time string
func getRelativeTime(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Parse the date string (format: 2006-01-02T15:04:05)
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// Try without the time part
		t, err = time.Parse("2006-01-02", dateStr[:10])
		if err != nil {
			return ""
		}
	}

	duration := time.Since(t)
	days := int(duration.Hours() / 24)
	weeks := days / 7

	if days < 1 {
		return "(< day ago)"
	} else if days == 1 {
		return "(1 day ago)"
	} else if weeks == 0 {
		return fmt.Sprintf("(%d days ago)", days)
	} else if weeks == 1 {
		return "(1 week ago)"
	} else if weeks <= 10 {
		return fmt.Sprintf("(%d weeks ago)", weeks)
	} else {
		return "(10+ weeks ago)"
	}
}
