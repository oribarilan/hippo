package main

import (
	"fmt"
	"testing"
)

// Helper function to create test work items
func createTestWorkItem(id int, title string, parentID *int) WorkItem {
	return WorkItem{
		ID:           id,
		Title:        title,
		ParentID:     parentID,
		State:        "Active",
		WorkItemType: "Task",
	}
}

// Helper to create test WorkItemList
func createTestList(tasks []WorkItem) *WorkItemList {
	return &WorkItemList{
		tasks:        tasks,
		cursor:       0,
		scrollOffset: 0,
		filterActive: false,
		loaded:       len(tasks),
		totalCount:   len(tasks),
		attempted:    true,
	}
}

// Helper to compare two TreeItem slices
func treeItemsEqual(a, b []TreeItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].WorkItem.ID != b[i].WorkItem.ID ||
			a[i].Depth != b[i].Depth ||
			len(a[i].IsLast) != len(b[i].IsLast) {
			return false
		}
	}
	return true
}

// TestBuildTreeStructure tests the tree building logic
func TestBuildTreeStructure(t *testing.T) {
	tests := []struct {
		name     string
		items    []WorkItem
		expected int // number of root items expected
	}{
		{
			name:     "Empty input",
			items:    []WorkItem{},
			expected: 0,
		},
		{
			name: "Single root item",
			items: []WorkItem{
				createTestWorkItem(1, "Task 1", nil),
			},
			expected: 1,
		},
		{
			name: "Multiple root items",
			items: []WorkItem{
				createTestWorkItem(1, "Task 1", nil),
				createTestWorkItem(2, "Task 2", nil),
			},
			expected: 2,
		},
		{
			name: "Parent-child relationship",
			items: []WorkItem{
				createTestWorkItem(1, "Parent", nil),
				createTestWorkItem(2, "Child", intPtr(1)),
			},
			expected: 1, // Only parent is root
		},
		{
			name: "Orphaned child (parent not in list)",
			items: []WorkItem{
				createTestWorkItem(2, "Child", intPtr(99)),
			},
			expected: 1, // Orphan treated as root
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := buildTreeStructure(tt.items)
			if len(roots) != tt.expected {
				t.Errorf("buildTreeStructure() returned %d roots, expected %d", len(roots), tt.expected)
			}

			// Verify parent-child relationships
			if tt.name == "Parent-child relationship" && len(roots) > 0 {
				if len(roots[0].Children) != 1 {
					t.Errorf("Expected parent to have 1 child, got %d", len(roots[0].Children))
				}
			}
		})
	}
}

// TestFlattenTree tests the tree flattening logic
func TestFlattenTree(t *testing.T) {
	tests := []struct {
		name          string
		items         []WorkItem
		expectedCount int
		expectedDepth []int // Expected depth for each flattened item
	}{
		{
			name:          "Empty tree",
			items:         []WorkItem{},
			expectedCount: 0,
			expectedDepth: []int{},
		},
		{
			name: "Single node",
			items: []WorkItem{
				createTestWorkItem(1, "Task 1", nil),
			},
			expectedCount: 1,
			expectedDepth: []int{0},
		},
		{
			name: "Parent with one child",
			items: []WorkItem{
				createTestWorkItem(1, "Parent", nil),
				createTestWorkItem(2, "Child", intPtr(1)),
			},
			expectedCount: 2,
			expectedDepth: []int{0, 1},
		},
		{
			name: "Nested hierarchy (3 levels)",
			items: []WorkItem{
				createTestWorkItem(1, "Root", nil),
				createTestWorkItem(2, "Child", intPtr(1)),
				createTestWorkItem(3, "Grandchild", intPtr(2)),
			},
			expectedCount: 3,
			expectedDepth: []int{0, 1, 2},
		},
		{
			name: "Multiple siblings",
			items: []WorkItem{
				createTestWorkItem(1, "Parent", nil),
				createTestWorkItem(2, "Child1", intPtr(1)),
				createTestWorkItem(3, "Child2", intPtr(1)),
			},
			expectedCount: 3,
			expectedDepth: []int{0, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := buildTreeStructure(tt.items)
			flattened := flattenTree(roots)

			if len(flattened) != tt.expectedCount {
				t.Errorf("flattenTree() returned %d items, expected %d", len(flattened), tt.expectedCount)
			}

			// Check depths
			for i, item := range flattened {
				if i < len(tt.expectedDepth) {
					if item.Depth != tt.expectedDepth[i] {
						t.Errorf("Item %d has depth %d, expected %d", i, item.Depth, tt.expectedDepth[i])
					}
				}
			}
		})
	}
}

// TestGetTreePrefix tests the tree prefix generation
func TestGetTreePrefix(t *testing.T) {
	tests := []struct {
		name     string
		depth    int
		isLast   []bool
		expected string
	}{
		{
			name:     "Root level",
			depth:    0,
			isLast:   []bool{},
			expected: "",
		},
		{
			name:     "First level child (last)",
			depth:    1,
			isLast:   []bool{true},
			expected: "╰── ",
		},
		{
			name:     "First level child (not last)",
			depth:    1,
			isLast:   []bool{false},
			expected: "├── ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := TreeItem{
				Depth:  tt.depth,
				IsLast: tt.isLast,
			}
			prefix := getTreePrefix(item)
			if prefix != tt.expected {
				t.Errorf("getTreePrefix() = %q, expected %q", prefix, tt.expected)
			}
		})
	}
}

// TestTreeCacheHit verifies cache works correctly
func TestTreeCacheHit(t *testing.T) {
	// Create test data
	tasks := []WorkItem{
		createTestWorkItem(1, "Task 1", nil),
		createTestWorkItem(2, "Task 2", intPtr(1)),
	}
	list := createTestList(tasks)

	// Build tree first time (cache miss)
	roots1 := buildTreeStructure(tasks)
	tree1 := flattenTree(roots1)
	list.treeCache = tree1

	// Second call should hit cache
	if list.treeCache == nil {
		t.Error("Cache should not be nil after first build")
	}

	// Verify cache contains expected items
	if len(list.treeCache) != 2 {
		t.Errorf("Cache has %d items, expected 2", len(list.treeCache))
	}
}

// TestCacheInvalidation tests that cache is properly invalidated
func TestCacheInvalidation(t *testing.T) {
	tasks := []WorkItem{
		createTestWorkItem(1, "Task 1", nil),
	}
	list := createTestList(tasks)

	// Build tree and populate cache
	roots := buildTreeStructure(tasks)
	list.treeCache = flattenTree(roots)
	initialVersion := list.cacheVersion

	// Invalidate cache
	list.invalidateTreeCache()

	// Verify cache is cleared and version incremented
	if list.treeCache != nil {
		t.Error("Cache should be nil after invalidation")
	}
	if list.cacheVersion != initialVersion+1 {
		t.Errorf("Cache version = %d, expected %d", list.cacheVersion, initialVersion+1)
	}
}

// TestCacheInvalidationOnReplaceTasks tests cache invalidation when replacing tasks
func TestCacheInvalidationOnReplaceTasks(t *testing.T) {
	tasks := []WorkItem{
		createTestWorkItem(1, "Task 1", nil),
	}
	list := createTestList(tasks)

	// Build tree and populate cache
	roots := buildTreeStructure(tasks)
	list.treeCache = flattenTree(roots)
	initialVersion := list.cacheVersion

	// Replace tasks (should invalidate cache)
	newTasks := []WorkItem{
		createTestWorkItem(2, "Task 2", nil),
	}
	list.replaceTasks(newTasks, 1)

	// Verify cache is invalidated
	if list.treeCache != nil {
		t.Error("Cache should be nil after replaceTasks")
	}
	if list.cacheVersion != initialVersion+1 {
		t.Errorf("Cache version = %d, expected %d", list.cacheVersion, initialVersion+1)
	}
}

// TestCacheInvalidationOnAppendTasks tests cache invalidation when appending tasks
func TestCacheInvalidationOnAppendTasks(t *testing.T) {
	tasks := []WorkItem{
		createTestWorkItem(1, "Task 1", nil),
	}
	list := createTestList(tasks)

	// Build tree and populate cache
	roots := buildTreeStructure(tasks)
	list.treeCache = flattenTree(roots)
	initialVersion := list.cacheVersion

	// Append tasks (should invalidate cache)
	newTasks := []WorkItem{
		createTestWorkItem(2, "Task 2", nil),
	}
	list.appendTasks(newTasks)

	// Verify cache is invalidated
	if list.treeCache != nil {
		t.Error("Cache should be nil after appendTasks")
	}
	if list.cacheVersion != initialVersion+1 {
		t.Errorf("Cache version = %d, expected %d", list.cacheVersion, initialVersion+1)
	}
	if len(list.tasks) != 2 {
		t.Errorf("List has %d tasks, expected 2", len(list.tasks))
	}
}

// TestTreeCacheWorkflow tests the complete workflow with caching
func TestTreeCacheWorkflow(t *testing.T) {
	// Create initial model with sprint lists
	m := model{
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),
		currentMode:  sprintMode,
		currentTab:   currentSprint,
	}

	// Create initial tasks
	tasks := []WorkItem{
		createTestWorkItem(1, "Parent", nil),
		createTestWorkItem(2, "Child", intPtr(1)),
	}

	// Initialize the list
	m.sprintLists[currentSprint] = createTestList(tasks)

	// First call to getVisibleTreeItems (cache miss)
	tree1 := m.getVisibleTreeItems()
	if len(tree1) != 2 {
		t.Errorf("First call returned %d items, expected 2", len(tree1))
	}

	list := m.getCurrentList()
	if list.treeCache == nil {
		t.Error("Cache should be populated after first call")
	}

	// Second call (cache hit - should return same structure)
	tree2 := m.getVisibleTreeItems()
	if len(tree2) != 2 {
		t.Errorf("Second call returned %d items, expected 2", len(tree2))
	}

	// Verify trees are identical
	if !treeItemsEqual(tree1, tree2) {
		t.Error("Trees should be identical on cache hit")
	}

	// Modify tasks (should invalidate cache)
	newTasks := []WorkItem{
		createTestWorkItem(1, "Parent Updated", nil),
		createTestWorkItem(2, "Child", intPtr(1)),
		createTestWorkItem(3, "New Child", intPtr(1)),
	}
	list.replaceTasks(newTasks, 3)

	// Next call should rebuild (cache miss)
	tree3 := m.getVisibleTreeItems()
	if len(tree3) != 3 {
		t.Errorf("After update, returned %d items, expected 3", len(tree3))
	}
}

// TestCacheAcrossMultipleLists tests that each list has independent cache
func TestCacheAcrossMultipleLists(t *testing.T) {
	m := model{
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),
		currentMode:  sprintMode,
		currentTab:   currentSprint,
	}

	// Create tasks for current sprint
	currentTasks := []WorkItem{
		createTestWorkItem(1, "Current Sprint Task", nil),
	}
	m.sprintLists[currentSprint] = createTestList(currentTasks)

	// Create tasks for next sprint
	nextTasks := []WorkItem{
		createTestWorkItem(2, "Next Sprint Task", nil),
		createTestWorkItem(3, "Next Sprint Task 2", nil),
	}
	m.sprintLists[nextSprint] = createTestList(nextTasks)

	// Build tree for current sprint
	tree1 := m.getVisibleTreeItems()
	if len(tree1) != 1 {
		t.Errorf("Current sprint tree has %d items, expected 1", len(tree1))
	}

	// Switch to next sprint
	m.currentTab = nextSprint

	// Build tree for next sprint
	tree2 := m.getVisibleTreeItems()
	if len(tree2) != 2 {
		t.Errorf("Next sprint tree has %d items, expected 2", len(tree2))
	}

	// Verify both caches exist independently
	currentList := m.sprintLists[currentSprint]
	nextList := m.sprintLists[nextSprint]

	if currentList.treeCache == nil {
		t.Error("Current sprint cache should exist")
	}
	if nextList.treeCache == nil {
		t.Error("Next sprint cache should exist")
	}
	if len(currentList.treeCache) != 1 {
		t.Errorf("Current sprint cache has %d items, expected 1", len(currentList.treeCache))
	}
	if len(nextList.treeCache) != 2 {
		t.Errorf("Next sprint cache has %d items, expected 2", len(nextList.treeCache))
	}
}

// Helper function to create an int pointer
func intPtr(i int) *int {
	return &i
}

// Benchmark tests

// BenchmarkBuildTreeStructure measures tree building performance with various sizes
func BenchmarkBuildTreeStructure(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_items", size), func(b *testing.B) {
			// Create test data
			tasks := make([]WorkItem, size)
			for i := 0; i < size; i++ {
				tasks[i] = createTestWorkItem(i+1, "Task", nil)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buildTreeStructure(tasks)
			}
		})
	}
}

// BenchmarkBuildTreeStructureWithHierarchy benchmarks tree building with nested items
func BenchmarkBuildTreeStructureWithHierarchy(b *testing.B) {
	// Create hierarchical data: 20 parents, each with 5 children
	tasks := make([]WorkItem, 120)
	idx := 0
	for p := 1; p <= 20; p++ {
		tasks[idx] = createTestWorkItem(p, "Parent", nil)
		idx++
		for c := 1; c <= 5; c++ {
			childID := p*100 + c
			tasks[idx] = createTestWorkItem(childID, "Child", intPtr(p))
			idx++
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildTreeStructure(tasks)
	}
}

// BenchmarkFlattenTree measures tree flattening performance
func BenchmarkFlattenTree(b *testing.B) {
	// Create test data: 50 parents, each with 2 children
	tasks := make([]WorkItem, 150)
	idx := 0
	for p := 1; p <= 50; p++ {
		tasks[idx] = createTestWorkItem(p, "Parent", nil)
		idx++
		for c := 1; c <= 2; c++ {
			childID := p*100 + c
			tasks[idx] = createTestWorkItem(childID, "Child", intPtr(p))
			idx++
		}
	}

	roots := buildTreeStructure(tasks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flattenTree(roots)
	}
}

// BenchmarkGetVisibleTreeItems_ColdCache measures first call performance (cache miss)
func BenchmarkGetVisibleTreeItems_ColdCache(b *testing.B) {
	// Create test data: 100 items with hierarchy
	tasks := make([]WorkItem, 100)
	for i := 0; i < 20; i++ {
		tasks[i] = createTestWorkItem(i+1, "Parent", nil)
	}
	for i := 20; i < 100; i++ {
		parentID := (i % 20) + 1
		tasks[i] = createTestWorkItem(i+1, "Child", intPtr(parentID))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create fresh model each iteration to simulate cold cache
		m := model{
			sprintLists:  make(map[sprintTab]*WorkItemList),
			backlogLists: make(map[backlogTab]*WorkItemList),
			currentMode:  sprintMode,
			currentTab:   currentSprint,
		}
		m.sprintLists[currentSprint] = createTestList(tasks)

		// First call - cache miss
		m.getVisibleTreeItems()
	}
}

// BenchmarkGetVisibleTreeItems_HotCache measures cached call performance (cache hit)
func BenchmarkGetVisibleTreeItems_HotCache(b *testing.B) {
	// Create test data: 100 items with hierarchy
	tasks := make([]WorkItem, 100)
	for i := 0; i < 20; i++ {
		tasks[i] = createTestWorkItem(i+1, "Parent", nil)
	}
	for i := 20; i < 100; i++ {
		parentID := (i % 20) + 1
		tasks[i] = createTestWorkItem(i+1, "Child", intPtr(parentID))
	}

	// Create model and warm up cache
	m := model{
		sprintLists:  make(map[sprintTab]*WorkItemList),
		backlogLists: make(map[backlogTab]*WorkItemList),
		currentMode:  sprintMode,
		currentTab:   currentSprint,
	}
	m.sprintLists[currentSprint] = createTestList(tasks)

	// Warm up cache
	m.getVisibleTreeItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Subsequent calls - cache hits
		m.getVisibleTreeItems()
	}
}

// BenchmarkTreeCacheVsNoCacheScrolling simulates scrolling scenario
func BenchmarkTreeCacheVsNoCacheScrolling(b *testing.B) {
	// Create realistic data: 150 items with hierarchy
	tasks := make([]WorkItem, 150)
	for i := 0; i < 30; i++ {
		tasks[i] = createTestWorkItem(i+1, "Parent", nil)
	}
	for i := 30; i < 150; i++ {
		parentID := (i % 30) + 1
		tasks[i] = createTestWorkItem(i+1, "Child", intPtr(parentID))
	}

	b.Run("With Cache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create model
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  sprintMode,
				currentTab:   currentSprint,
			}
			m.sprintLists[currentSprint] = createTestList(tasks)

			// First call to populate cache
			m.getVisibleTreeItems()

			// Simulate 100 render calls (like scrolling)
			for j := 0; j < 100; j++ {
				m.getVisibleTreeItems()
			}
		}
	})

	b.Run("Without Cache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create model
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  sprintMode,
				currentTab:   currentSprint,
			}
			m.sprintLists[currentSprint] = createTestList(tasks)

			// Simulate 100 render calls WITHOUT cache (rebuild every time)
			for j := 0; j < 100; j++ {
				// Invalidate cache to force rebuild
				if list := m.getCurrentList(); list != nil {
					list.invalidateTreeCache()
				}
				m.getVisibleTreeItems()
			}
		}
	})
}
