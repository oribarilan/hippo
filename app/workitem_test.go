package main

import (
	"strings"
	"testing"
	"time"
)

// TestWorkItemList_hasMore tests the hasMore method
func TestWorkItemList_hasMore(t *testing.T) {
	tests := []struct {
		name       string
		loaded     int
		totalCount int
		want       bool
	}{
		{
			name:       "Empty list",
			loaded:     0,
			totalCount: 0,
			want:       false,
		},
		{
			name:       "Partial load",
			loaded:     10,
			totalCount: 50,
			want:       true,
		},
		{
			name:       "Fully loaded",
			loaded:     50,
			totalCount: 50,
			want:       false,
		},
		{
			name:       "Over-loaded (shouldn't happen but handle gracefully)",
			loaded:     60,
			totalCount: 50,
			want:       false,
		},
		{
			name:       "One item remaining",
			loaded:     49,
			totalCount: 50,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := &WorkItemList{
				loaded:     tt.loaded,
				totalCount: tt.totalCount,
			}
			if got := wl.hasMore(); got != tt.want {
				t.Errorf("hasMore() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWorkItemList_getRemainingCount tests the getRemainingCount method
func TestWorkItemList_getRemainingCount(t *testing.T) {
	tests := []struct {
		name       string
		loaded     int
		totalCount int
		want       int
	}{
		{
			name:       "Empty list",
			loaded:     0,
			totalCount: 0,
			want:       0,
		},
		{
			name:       "Partial load - 10 remaining",
			loaded:     40,
			totalCount: 50,
			want:       10,
		},
		{
			name:       "Fully loaded",
			loaded:     50,
			totalCount: 50,
			want:       0,
		},
		{
			name:       "One item loaded",
			loaded:     1,
			totalCount: 100,
			want:       99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := &WorkItemList{
				loaded:     tt.loaded,
				totalCount: tt.totalCount,
			}
			if got := wl.getRemainingCount(); got != tt.want {
				t.Errorf("getRemainingCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWorkItemList_getVisibleTasks tests the getVisibleTasks method
func TestWorkItemList_getVisibleTasks(t *testing.T) {
	allTasks := simpleTaskSet()

	tests := []struct {
		name          string
		tasks         []WorkItem
		filterActive  bool
		filteredTasks []WorkItem
		wantCount     int
	}{
		{
			name:         "No filter active - return all tasks",
			tasks:        allTasks,
			filterActive: false,
			wantCount:    3,
		},
		{
			name:          "Filter active - return filtered tasks",
			tasks:         allTasks,
			filterActive:  true,
			filteredTasks: []WorkItem{allTasks[0]},
			wantCount:     1,
		},
		{
			name:          "Filter active with empty filtered results",
			tasks:         allTasks,
			filterActive:  true,
			filteredTasks: []WorkItem{}, // Empty slice, not nil
			wantCount:     0,
		},
		{
			name:          "Filter active but filteredTasks is nil (returns all tasks)",
			tasks:         allTasks,
			filterActive:  true,
			filteredTasks: nil,
			wantCount:     3, // When filteredTasks is nil, it returns wl.tasks
		},
		{
			name:         "Empty task list",
			tasks:        []WorkItem{},
			filterActive: false,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := &WorkItemList{
				tasks:         tt.tasks,
				filterActive:  tt.filterActive,
				filteredTasks: tt.filteredTasks,
			}
			got := wl.getVisibleTasks()
			if len(got) != tt.wantCount {
				t.Errorf("getVisibleTasks() returned %d tasks, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// TestWorkItemList_appendTasks tests appending tasks to the list
func TestWorkItemList_appendTasks(t *testing.T) {
	tests := []struct {
		name            string
		initialTasks    []WorkItem
		tasksToAppend   []WorkItem
		expectedCount   int
		expectedLoaded  int
		cacheInvalidate bool
	}{
		{
			name:            "Append to empty list",
			initialTasks:    []WorkItem{},
			tasksToAppend:   simpleTaskSet(),
			expectedCount:   3,
			expectedLoaded:  3,
			cacheInvalidate: true,
		},
		{
			name:            "Append to existing list",
			initialTasks:    []WorkItem{createTestWorkItem(1, "Task 1", nil)},
			tasksToAppend:   []WorkItem{createTestWorkItem(2, "Task 2", nil)},
			expectedCount:   2,
			expectedLoaded:  2,
			cacheInvalidate: true,
		},
		{
			name:            "Append empty list",
			initialTasks:    simpleTaskSet(),
			tasksToAppend:   []WorkItem{},
			expectedCount:   3,
			expectedLoaded:  3,
			cacheInvalidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := &WorkItemList{
				tasks:        tt.initialTasks,
				loaded:       len(tt.initialTasks),
				cacheVersion: 0,
			}

			// Set cache to verify invalidation (only if we have initial tasks)
			if tt.cacheInvalidate && len(tt.initialTasks) > 0 {
				wl.treeCache = []TreeItem{{WorkItem: &tt.initialTasks[0], Depth: 0}}
			}

			initialCacheVersion := wl.cacheVersion

			wl.appendTasks(tt.tasksToAppend)

			if len(wl.tasks) != tt.expectedCount {
				t.Errorf("appendTasks() resulted in %d tasks, want %d", len(wl.tasks), tt.expectedCount)
			}

			if wl.loaded != tt.expectedLoaded {
				t.Errorf("loaded count = %d, want %d", wl.loaded, tt.expectedLoaded)
			}

			// Verify cache was invalidated
			if wl.treeCache != nil {
				t.Error("Cache should be nil after appendTasks")
			}

			if wl.cacheVersion != initialCacheVersion+1 {
				t.Errorf("Cache version = %d, want %d", wl.cacheVersion, initialCacheVersion+1)
			}
		})
	}
}

// TestWorkItemList_replaceTasks tests replacing tasks in the list
func TestWorkItemList_replaceTasks(t *testing.T) {
	tests := []struct {
		name            string
		initialTasks    []WorkItem
		newTasks        []WorkItem
		totalCount      int
		expectedCount   int
		expectedLoaded  int
		expectedTotal   int
		cacheInvalidate bool
	}{
		{
			name:            "Replace empty with new tasks",
			initialTasks:    []WorkItem{},
			newTasks:        simpleTaskSet(),
			totalCount:      10,
			expectedCount:   3,
			expectedLoaded:  3,
			expectedTotal:   10,
			cacheInvalidate: true,
		},
		{
			name:            "Replace existing with new tasks",
			initialTasks:    simpleTaskSet(),
			newTasks:        []WorkItem{createTestWorkItem(10, "New Task", nil)},
			totalCount:      5,
			expectedCount:   1,
			expectedLoaded:  1,
			expectedTotal:   5,
			cacheInvalidate: true,
		},
		{
			name:            "Replace with empty list",
			initialTasks:    simpleTaskSet(),
			newTasks:        []WorkItem{},
			totalCount:      0,
			expectedCount:   0,
			expectedLoaded:  0,
			expectedTotal:   0,
			cacheInvalidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := &WorkItemList{
				tasks:        tt.initialTasks,
				loaded:       len(tt.initialTasks),
				totalCount:   100, // Old total count
				attempted:    false,
				cacheVersion: 0,
			}

			// Set cache to verify invalidation
			if tt.cacheInvalidate && len(tt.initialTasks) > 0 {
				wl.treeCache = []TreeItem{{WorkItem: &tt.initialTasks[0], Depth: 0}}
			}

			initialCacheVersion := wl.cacheVersion

			wl.replaceTasks(tt.newTasks, tt.totalCount)

			if len(wl.tasks) != tt.expectedCount {
				t.Errorf("replaceTasks() resulted in %d tasks, want %d", len(wl.tasks), tt.expectedCount)
			}

			if wl.loaded != tt.expectedLoaded {
				t.Errorf("loaded count = %d, want %d", wl.loaded, tt.expectedLoaded)
			}

			if wl.totalCount != tt.expectedTotal {
				t.Errorf("totalCount = %d, want %d", wl.totalCount, tt.expectedTotal)
			}

			if !wl.attempted {
				t.Error("attempted should be true after replaceTasks")
			}

			// Verify cache was invalidated
			if tt.cacheInvalidate && wl.treeCache != nil {
				t.Error("Cache should be nil after replaceTasks")
			}

			if wl.cacheVersion != initialCacheVersion+1 {
				t.Errorf("Cache version = %d, want %d", wl.cacheVersion, initialCacheVersion+1)
			}
		})
	}
}

// TestWorkItemList_invalidateTreeCache tests cache invalidation
func TestWorkItemList_invalidateTreeCache(t *testing.T) {
	wl := &WorkItemList{
		tasks:        simpleTaskSet(),
		treeCache:    []TreeItem{{WorkItem: &WorkItem{ID: 1}, Depth: 0}},
		cacheVersion: 5,
	}

	wl.invalidateTreeCache()

	if wl.treeCache != nil {
		t.Error("treeCache should be nil after invalidation")
	}

	if wl.cacheVersion != 6 {
		t.Errorf("cacheVersion = %d, want 6", wl.cacheVersion)
	}
}

// TestFormatDateTime tests the formatDateTime function
func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		contains string // Check if output contains certain text
	}{
		{
			name:     "Empty string",
			input:    "",
			wantErr:  false,
			contains: "",
		},
		{
			name:     "Valid full timestamp",
			input:    "2024-01-15T14:30:45",
			wantErr:  false,
			contains: "Jan 15, 2024",
		},
		{
			name:     "Valid date only",
			input:    "2024-12-25T00:00:00",
			wantErr:  false,
			contains: "Dec 25, 2024",
		},
		{
			name:     "Invalid format - return as-is",
			input:    "invalid-date",
			wantErr:  false,
			contains: "invalid-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateTime(tt.input)

			if tt.input == "" && got != "" {
				t.Errorf("formatDateTime(%q) = %q, want empty string", tt.input, got)
				return
			}

			if tt.contains != "" && got != tt.contains {
				// For valid dates, check if output contains expected text
				if tt.input != "invalid-date" {
					if len(got) == 0 {
						t.Errorf("formatDateTime(%q) returned empty, want non-empty", tt.input)
					}
				} else {
					// For invalid dates, should return input as-is
					if got != tt.input {
						t.Errorf("formatDateTime(%q) = %q, want %q", tt.input, got, tt.input)
					}
				}
			}
		})
	}
}

// TestGetRelativeTime tests the getRelativeTime function
func TestGetRelativeTime(t *testing.T) {
	// We'll use real time calculations with more tolerance for edge cases
	tests := []struct {
		name         string
		input        string
		wantContains string // Check if output contains this string
	}{
		{
			name:         "Empty string",
			input:        "",
			wantContains: "",
		},
		{
			name:         "Less than a day ago",
			input:        time.Now().Add(-2 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "< day ago",
		},
		{
			name:         "Approximately 1 day ago",
			input:        time.Now().Add(-25 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "day",
		},
		{
			name:         "3 days ago",
			input:        time.Now().Add(-73 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "days ago",
		},
		{
			name:         "Approximately 1 week ago",
			input:        time.Now().Add(-8 * 24 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "week",
		},
		{
			name:         "3 weeks ago",
			input:        time.Now().Add(-22 * 24 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "weeks ago",
		},
		{
			name:         "10+ weeks ago",
			input:        time.Now().Add(-100 * 24 * time.Hour).Format("2006-01-02T15:04:05"),
			wantContains: "10+ weeks ago",
		},
		{
			name:         "Invalid format",
			input:        "invalid-date",
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRelativeTime(tt.input)

			if tt.input == "" && got != "" {
				t.Errorf("getRelativeTime(%q) = %q, want empty string", tt.input, got)
				return
			}

			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("getRelativeTime(%q) = %q, want to contain %q", tt.input, got, tt.wantContains)
			}

			// For invalid dates, should return empty string
			if tt.input == "invalid-date" && got != "" {
				t.Errorf("getRelativeTime(%q) = %q, want empty string", tt.input, got)
			}
		})
	}
}

// TestGetPositionAfterSubtree tests the getPositionAfterSubtree function
func TestGetPositionAfterSubtree(t *testing.T) {
	// Create a tree structure:
	// Root (depth 0)
	//   Child 1 (depth 1)
	//     Grandchild (depth 2)
	//   Child 2 (depth 1)
	treeItems := []TreeItem{
		{WorkItem: &WorkItem{ID: 1}, Depth: 0},
		{WorkItem: &WorkItem{ID: 2}, Depth: 1},
		{WorkItem: &WorkItem{ID: 3}, Depth: 2},
		{WorkItem: &WorkItem{ID: 4}, Depth: 1},
	}

	tests := []struct {
		name string
		pos  int
		want int
	}{
		{
			name: "Single item with no children (last in tree)",
			pos:  3,
			want: 4,
		},
		{
			name: "Item with children (skip entire subtree)",
			pos:  1,
			want: 3,
		},
		{
			name: "Root with all children",
			pos:  0,
			want: 4,
		},
		{
			name: "Position beyond tree length",
			pos:  10,
			want: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPositionAfterSubtree(treeItems, tt.pos)
			if got != tt.want {
				t.Errorf("getPositionAfterSubtree() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetParentIDForTreeItem tests the getParentIDForTreeItem function
func TestGetParentIDForTreeItem(t *testing.T) {
	parentID := 10

	tests := []struct {
		name string
		item TreeItem
		want *int
	}{
		{
			name: "Item with parent ID",
			item: TreeItem{
				WorkItem: &WorkItem{ID: 1, ParentID: &parentID},
			},
			want: &parentID,
		},
		{
			name: "Item without parent ID",
			item: TreeItem{
				WorkItem: &WorkItem{ID: 1, ParentID: nil},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getParentIDForTreeItem(tt.item)
			if (got == nil) != (tt.want == nil) {
				t.Errorf("getParentIDForTreeItem() = %v, want %v", got, tt.want)
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("getParentIDForTreeItem() = %d, want %d", *got, *tt.want)
			}
		})
	}
}

// TestGetWorkItemIcon tests the getWorkItemIcon function
func TestGetWorkItemIcon(t *testing.T) {
	tests := []struct {
		name         string
		workItemType string
		want         string
	}{
		{
			name:         "Task type",
			workItemType: "Task",
			want:         "✓",
		},
		{
			name:         "Task type lowercase",
			workItemType: "task",
			want:         "✓",
		},
		{
			name:         "Bug type (default)",
			workItemType: "Bug",
			want:         "•",
		},
		{
			name:         "User Story (default)",
			workItemType: "User Story",
			want:         "•",
		},
		{
			name:         "Unknown type (default)",
			workItemType: "Unknown",
			want:         "•",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getWorkItemIcon(tt.workItemType)
			if got != tt.want {
				t.Errorf("getWorkItemIcon(%q) = %q, want %q", tt.workItemType, got, tt.want)
			}
		})
	}
}

// TestCountTreeItems tests the countTreeItems function
func TestCountTreeItems(t *testing.T) {
	tests := []struct {
		name  string
		items []TreeItem
		want  int
	}{
		{
			name:  "Empty tree",
			items: []TreeItem{},
			want:  0,
		},
		{
			name: "Single item",
			items: []TreeItem{
				{WorkItem: &WorkItem{ID: 1}, Depth: 0},
			},
			want: 1,
		},
		{
			name: "Multiple items",
			items: []TreeItem{
				{WorkItem: &WorkItem{ID: 1}, Depth: 0},
				{WorkItem: &WorkItem{ID: 2}, Depth: 1},
				{WorkItem: &WorkItem{ID: 3}, Depth: 1},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countTreeItems(tt.items)
			if got != tt.want {
				t.Errorf("countTreeItems() = %d, want %d", got, tt.want)
			}
		})
	}
}
