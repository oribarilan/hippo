package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

// TestGetCurrentList tests the getCurrentList method
func TestGetCurrentList(t *testing.T) {
	tests := []struct {
		name            string
		currentMode     appMode
		currentTab      sprintTab
		currentBacklog  backlogTab
		setupLists      func(m *model)
		expectNonNil    bool
		expectedTaskLen int
	}{
		{
			name:        "Sprint mode - current sprint exists",
			currentMode: sprintMode,
			currentTab:  currentSprint,
			setupLists: func(m *model) {
				m.sprintLists[currentSprint] = createTestList(simpleTaskSet())
			},
			expectNonNil:    true,
			expectedTaskLen: 3,
		},
		{
			name:         "Sprint mode - list doesn't exist (returns empty)",
			currentMode:  sprintMode,
			currentTab:   nextSprint,
			setupLists:   func(m *model) {},
			expectNonNil: true, // Returns empty list, not nil
		},
		{
			name:           "Backlog mode - recent backlog exists",
			currentMode:    backlogMode,
			currentBacklog: recentBacklog,
			setupLists: func(m *model) {
				m.backlogLists[recentBacklog] = createTestList(simpleTaskSet())
			},
			expectNonNil:    true,
			expectedTaskLen: 3,
		},
		{
			name:           "Backlog mode - list doesn't exist (returns empty)",
			currentMode:    backlogMode,
			currentBacklog: abandonedWork,
			setupLists:     func(m *model) {},
			expectNonNil:   true, // Returns empty list, not nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:       make(map[sprintTab]*WorkItemList),
				backlogLists:      make(map[backlogTab]*WorkItemList),
				currentMode:       tt.currentMode,
				currentTab:        tt.currentTab,
				currentBacklogTab: tt.currentBacklog,
			}

			tt.setupLists(&m)

			list := m.getCurrentList()

			if !tt.expectNonNil && list != nil {
				t.Error("Expected nil, got non-nil list")
			}

			if tt.expectNonNil && list == nil {
				t.Error("Expected non-nil list, got nil")
			}

			if tt.expectedTaskLen > 0 && len(list.tasks) != tt.expectedTaskLen {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTaskLen, len(list.tasks))
			}
		})
	}
}

// TestEnsureCurrentListExists tests the ensureCurrentListExists method
func TestEnsureCurrentListExists(t *testing.T) {
	tests := []struct {
		name           string
		currentMode    appMode
		currentTab     sprintTab
		currentBacklog backlogTab
		setupLists     func(m *model)
		checkList      func(m *model) bool
	}{
		{
			name:        "Sprint mode - create missing list",
			currentMode: sprintMode,
			currentTab:  currentSprint,
			setupLists:  func(m *model) {},
			checkList: func(m *model) bool {
				_, exists := m.sprintLists[currentSprint]
				return exists
			},
		},
		{
			name:        "Sprint mode - list already exists",
			currentMode: sprintMode,
			currentTab:  currentSprint,
			setupLists: func(m *model) {
				m.sprintLists[currentSprint] = createTestList(simpleTaskSet())
			},
			checkList: func(m *model) bool {
				list, exists := m.sprintLists[currentSprint]
				return exists && len(list.tasks) == 3 // Original tasks preserved
			},
		},
		{
			name:           "Backlog mode - create missing list",
			currentMode:    backlogMode,
			currentBacklog: recentBacklog,
			setupLists:     func(m *model) {},
			checkList: func(m *model) bool {
				_, exists := m.backlogLists[recentBacklog]
				return exists
			},
		},
		{
			name:           "Backlog mode - list already exists",
			currentMode:    backlogMode,
			currentBacklog: abandonedWork,
			setupLists: func(m *model) {
				m.backlogLists[abandonedWork] = createTestList(simpleTaskSet())
			},
			checkList: func(m *model) bool {
				list, exists := m.backlogLists[abandonedWork]
				return exists && len(list.tasks) == 3 // Original tasks preserved
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:       make(map[sprintTab]*WorkItemList),
				backlogLists:      make(map[backlogTab]*WorkItemList),
				currentMode:       tt.currentMode,
				currentTab:        tt.currentTab,
				currentBacklogTab: tt.currentBacklog,
			}

			tt.setupLists(&m)
			m.ensureCurrentListExists()

			if !tt.checkList(&m) {
				t.Error("List check failed")
			}
		})
	}
}

// TestGetCurrentTasks tests the getCurrentTasks method
func TestGetCurrentTasks(t *testing.T) {
	tasks := simpleTaskSet()

	tests := []struct {
		name        string
		setupModel  func() model
		expectedLen int
		expectedNil bool
	}{
		{
			name: "Sprint mode with tasks",
			setupModel: func() model {
				m := model{
					sprintLists:  make(map[sprintTab]*WorkItemList),
					backlogLists: make(map[backlogTab]*WorkItemList),
					currentMode:  sprintMode,
					currentTab:   currentSprint,
				}
				m.sprintLists[currentSprint] = createTestList(tasks)
				return m
			},
			expectedLen: 3,
		},
		{
			name: "Sprint mode with no list",
			setupModel: func() model {
				return model{
					sprintLists:  make(map[sprintTab]*WorkItemList),
					backlogLists: make(map[backlogTab]*WorkItemList),
					currentMode:  sprintMode,
					currentTab:   currentSprint,
				}
			},
			expectedLen: 0,
		},
		{
			name: "Backlog mode with tasks",
			setupModel: func() model {
				m := model{
					sprintLists:       make(map[sprintTab]*WorkItemList),
					backlogLists:      make(map[backlogTab]*WorkItemList),
					currentMode:       backlogMode,
					currentBacklogTab: recentBacklog,
				}
				m.backlogLists[recentBacklog] = createTestList(tasks)
				return m
			},
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			got := m.getCurrentTasks()

			if len(got) != tt.expectedLen {
				t.Errorf("getCurrentTasks() returned %d tasks, want %d", len(got), tt.expectedLen)
			}
		})
	}
}

// TestSetCurrentTasks tests the setCurrentTasks method
func TestSetCurrentTasks(t *testing.T) {
	newTasks := simpleTaskSet()

	tests := []struct {
		name        string
		currentMode appMode
		currentTab  sprintTab
		expectedLen int
	}{
		{
			name:        "Sprint mode - set tasks",
			currentMode: sprintMode,
			currentTab:  currentSprint,
			expectedLen: 3,
		},
		{
			name:        "Sprint mode - set empty tasks",
			currentMode: sprintMode,
			currentTab:  previousSprint,
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  tt.currentMode,
				currentTab:   tt.currentTab,
			}

			m.setCurrentTasks(newTasks)

			// Verify list was created and tasks were set
			list := m.getCurrentList()
			if list == nil {
				t.Fatal("List should not be nil after setCurrentTasks")
			}

			if len(list.tasks) != tt.expectedLen {
				t.Errorf("List has %d tasks, want %d", len(list.tasks), tt.expectedLen)
			}

			if list.loaded != tt.expectedLen {
				t.Errorf("loaded count = %d, want %d", list.loaded, tt.expectedLen)
			}
		})
	}
}

// TestFilterSearch tests the filterSearch method
func TestFilterSearch(t *testing.T) {
	allTasks := []WorkItem{
		createTestWorkItem(1, "Buy groceries", nil),
		createTestWorkItem(2, "Fix bug in login", nil),
		createTestWorkItem(3, "Write documentation", nil),
	}

	tests := []struct {
		name          string
		filterInput   string
		expectedCount int
		shouldFilter  bool
	}{
		{
			name:          "Empty query - no filter",
			filterInput:   "",
			expectedCount: 0, // filteredTasks should be nil
			shouldFilter:  false,
		},
		{
			name:          "Search by title - partial match",
			filterInput:   "bug",
			expectedCount: 1,
			shouldFilter:  true,
		},
		{
			name:          "Search by ID",
			filterInput:   "2",
			expectedCount: 1,
			shouldFilter:  true,
		},
		{
			name:          "Search by title - case insensitive",
			filterInput:   "BUY",
			expectedCount: 1,
			shouldFilter:  true,
		},
		{
			name:          "Search with no matches",
			filterInput:   "nonexistent",
			expectedCount: 0,
			shouldFilter:  true,
		},
		{
			name:          "Search matches multiple",
			filterInput:   "i", // matches "Buy", "Fix", "Write"
			expectedCount: 3,
			shouldFilter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  sprintMode,
				currentTab:   currentSprint,
				filter: FilterState{
					filterInput: textinput.Model{},
				},
			}

			// Initialize list with tasks
			m.sprintLists[currentSprint] = createTestList(allTasks)

			// Set filter input value
			m.filter.filterInput.SetValue(tt.filterInput)

			// Perform filter
			m.filterSearch()

			list := m.getCurrentList()
			if list == nil {
				t.Fatal("List should not be nil")
			}

			if list.filterActive != tt.shouldFilter {
				t.Errorf("filterActive = %v, want %v", list.filterActive, tt.shouldFilter)
			}

			if tt.shouldFilter {
				if list.filteredTasks == nil && tt.expectedCount > 0 {
					t.Error("filteredTasks should not be nil when filter is active")
				}
				if list.filteredTasks != nil && len(list.filteredTasks) != tt.expectedCount {
					t.Errorf("filteredTasks has %d items, want %d", len(list.filteredTasks), tt.expectedCount)
				}
			} else {
				if list.filteredTasks != nil {
					t.Error("filteredTasks should be nil when filter is not active")
				}
			}

			// Cache should be invalidated
			if list.treeCache != nil {
				t.Error("Cache should be invalidated after filter change")
			}
		})
	}
}

// TestGetVisibleTasks tests the getVisibleTasks method (model-level)
func TestGetVisibleTasks_ModelLevel(t *testing.T) {
	allTasks := []WorkItem{
		createTestWorkItemWithIteration(1, "Sprint 1 Task", "Project\\Sprint 1"),
		createTestWorkItemWithIteration(2, "Sprint 2 Task", "Project\\Sprint 2"),
		createTestWorkItemWithIteration(3, "No Sprint Task", ""),
	}

	tests := []struct {
		name          string
		mode          appMode
		setupModel    func(m *model)
		expectedCount int
		expectedIDs   []int
	}{
		{
			name: "Sprint mode - filter by sprint path",
			mode: sprintMode,
			setupModel: func(m *model) {
				m.currentTab = currentSprint
				m.sprints = map[sprintTab]*Sprint{
					currentSprint: createTestSprint("Sprint 1", "Project\\Sprint 1", "2024-01-01", "2024-01-14"),
				}
				m.sprintLists[currentSprint] = createTestList(allTasks)
			},
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
		{
			name: "Sprint mode - no sprint data (show all)",
			mode: sprintMode,
			setupModel: func(m *model) {
				m.currentTab = currentSprint
				m.sprints = map[sprintTab]*Sprint{}
				m.sprintLists[currentSprint] = createTestList(allTasks)
			},
			expectedCount: 3,
		},
		{
			name: "Backlog mode - show all tasks",
			mode: backlogMode,
			setupModel: func(m *model) {
				m.currentBacklogTab = recentBacklog
				m.backlogLists[recentBacklog] = createTestList(allTasks)
			},
			expectedCount: 3,
		},
		{
			name: "Sprint mode with filter active",
			mode: sprintMode,
			setupModel: func(m *model) {
				m.currentTab = currentSprint
				m.sprints = map[sprintTab]*Sprint{
					currentSprint: createTestSprint("Sprint 1", "Project\\Sprint 1", "2024-01-01", "2024-01-14"),
				}
				list := createTestList(allTasks)
				// Set filter active
				list.filterActive = true
				list.filteredTasks = []WorkItem{allTasks[0]} // Only Sprint 1 task
				m.sprintLists[currentSprint] = list
			},
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  tt.mode,
				sprints:      make(map[sprintTab]*Sprint),
			}

			tt.setupModel(&m)

			got := m.getVisibleTasks()

			if len(got) != tt.expectedCount {
				t.Errorf("getVisibleTasks() returned %d tasks, want %d", len(got), tt.expectedCount)
			}

			if tt.expectedIDs != nil {
				for i, id := range tt.expectedIDs {
					if i >= len(got) {
						t.Errorf("Missing expected task ID %d at index %d", id, i)
						continue
					}
					if got[i].ID != id {
						t.Errorf("Task at index %d has ID %d, want %d", i, got[i].ID, id)
					}
				}
			}
		})
	}
}

// TestGetParentTask tests the getParentTask method
func TestGetParentTask(t *testing.T) {
	parentTask := createTestWorkItem(1, "Parent Task", nil)
	childTask := createTestWorkItem(2, "Child Task", intPtr(1))
	orphanTask := createTestWorkItem(3, "Orphan Task", intPtr(999))

	tests := []struct {
		name       string
		task       *WorkItem
		allTasks   []WorkItem
		wantParent bool
		wantID     int
	}{
		{
			name:       "Task with existing parent",
			task:       &childTask,
			allTasks:   []WorkItem{parentTask, childTask},
			wantParent: true,
			wantID:     1,
		},
		{
			name:       "Task with missing parent",
			task:       &orphanTask,
			allTasks:   []WorkItem{parentTask, orphanTask},
			wantParent: false,
		},
		{
			name:       "Task with no parent ID",
			task:       &parentTask,
			allTasks:   []WorkItem{parentTask},
			wantParent: false,
		},
		{
			name:       "Nil task",
			task:       nil,
			allTasks:   []WorkItem{parentTask},
			wantParent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  sprintMode,
				currentTab:   currentSprint,
			}

			m.sprintLists[currentSprint] = createTestList(tt.allTasks)

			got := m.getParentTask(tt.task)

			if tt.wantParent && got == nil {
				t.Error("Expected parent task, got nil")
			}

			if !tt.wantParent && got != nil {
				t.Errorf("Expected nil, got parent task with ID %d", got.ID)
			}

			if tt.wantParent && got != nil && got.ID != tt.wantID {
				t.Errorf("Parent task ID = %d, want %d", got.ID, tt.wantID)
			}
		})
	}
}

// TestGetContentHeight tests the getContentHeight method
func TestGetContentHeight(t *testing.T) {
	tests := []struct {
		name          string
		height        int
		hint          string
		statusMessage string
		minExpected   int
	}{
		{
			name:        "Normal height with no extras",
			height:      50,
			hint:        "",
			minExpected: 5, // Should be much higher, but at least minimum
		},
		{
			name:        "Small terminal",
			height:      10,
			hint:        "",
			minExpected: 5, // Minimum content height
		},
		{
			name:        "With hint",
			height:      50,
			hint:        "Sprint: 2024-01-01 to 2024-01-14",
			minExpected: 5,
		},
		{
			name:          "With status message",
			height:        50,
			statusMessage: "Task updated successfully",
			minExpected:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:   make(map[sprintTab]*WorkItemList),
				backlogLists:  make(map[backlogTab]*WorkItemList),
				currentMode:   sprintMode,
				currentTab:    currentSprint,
				statusMessage: tt.statusMessage,
				sprints:       make(map[sprintTab]*Sprint),
				ui: UIState{
					height: tt.height,
				},
			}

			if tt.hint != "" {
				m.sprints[currentSprint] = &Sprint{
					StartDate: "2024-01-01",
					EndDate:   "2024-01-14",
				}
			}

			got := m.getContentHeight()

			if got < tt.minExpected {
				t.Errorf("getContentHeight() = %d, want at least %d", got, tt.minExpected)
			}

			// Content height should never exceed terminal height
			if got > tt.height {
				t.Errorf("getContentHeight() = %d exceeds terminal height %d", got, tt.height)
			}
		})
	}
}

// TestAdjustScrollOffset tests the adjustScrollOffset method
func TestAdjustScrollOffset(t *testing.T) {
	tests := []struct {
		name              string
		cursor            int
		scrollOffset      int
		terminalHeight    int
		setupModel        func(m *model) // Setup function to configure model for desired contentHeight
		expectedScrollMin int
		expectedScrollMax int
	}{
		{
			name:           "Cursor above visible area - scroll up",
			cursor:         5,
			scrollOffset:   10,
			terminalHeight: 50,
			setupModel: func(m *model) {
				// Setup to get reasonable content height
				m.currentMode = sprintMode
				m.currentTab = currentSprint
				m.sprints = make(map[sprintTab]*Sprint)
			},
			expectedScrollMin: 5,
			expectedScrollMax: 5,
		},
		{
			name:           "Cursor below visible area - scroll down",
			cursor:         30,
			scrollOffset:   0,
			terminalHeight: 30,
			setupModel: func(m *model) {
				m.currentMode = sprintMode
				m.currentTab = currentSprint
				m.sprints = make(map[sprintTab]*Sprint)
			},
			// With terminal height 30, fixed height = 10, content height = 20
			// Cursor at 30, scroll at 0 means cursor is at position 30
			// Condition: 30 >= 0 + 20 (TRUE, cursor is below visible area)
			// New scroll = 30 - 20 + 1 = 11
			expectedScrollMin: 11,
			expectedScrollMax: 11,
		},
		{
			name:           "Cursor in visible area - no scroll",
			cursor:         15,
			scrollOffset:   10,
			terminalHeight: 50,
			setupModel: func(m *model) {
				m.currentMode = sprintMode
				m.currentTab = currentSprint
				m.sprints = make(map[sprintTab]*Sprint)
			},
			// Cursor 15, scroll 10, if content height > 5, cursor is visible
			expectedScrollMin: 10,
			expectedScrollMax: 10,
		},
		{
			name:           "Negative scroll offset correction",
			cursor:         0,
			scrollOffset:   -5,
			terminalHeight: 40,
			setupModel: func(m *model) {
				m.currentMode = sprintMode
				m.currentTab = currentSprint
				m.sprints = make(map[sprintTab]*Sprint)
			},
			expectedScrollMin: 0,
			expectedScrollMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				ui: UIState{
					cursor:       tt.cursor,
					scrollOffset: tt.scrollOffset,
					height:       tt.terminalHeight,
				},
			}

			if tt.setupModel != nil {
				tt.setupModel(&m)
			}

			m.adjustScrollOffset()

			if m.ui.scrollOffset < tt.expectedScrollMin || m.ui.scrollOffset > tt.expectedScrollMax {
				t.Errorf("scrollOffset = %d, want between %d and %d", m.ui.scrollOffset, tt.expectedScrollMin, tt.expectedScrollMax)
			}

			// Scroll offset should never be negative
			if m.ui.scrollOffset < 0 {
				t.Errorf("scrollOffset = %d, should never be negative", m.ui.scrollOffset)
			}
		})
	}
}

// TestGetStateCategory tests the getStateCategory method
func TestGetStateCategory(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "Known state - exact match",
			state:    "Active",
			expected: "InProgress", // Assuming Active maps to InProgress
		},
		{
			name:     "Closed state - fallback",
			state:    "Closed",
			expected: "Completed",
		},
		{
			name:     "Done state - fallback",
			state:    "Done",
			expected: "Completed",
		},
		{
			name:     "New state - fallback",
			state:    "New",
			expected: "Proposed",
		},
		{
			name:     "Unknown state - default to InProgress",
			state:    "UnknownState",
			expected: "InProgress",
		},
		{
			name:     "Case insensitive - CLOSED",
			state:    "CLOSED",
			expected: "Completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				stateCategories: map[string]string{
					"Active": "InProgress",
					"Done":   "Completed",
				},
			}

			got := m.getStateCategory(tt.state)

			if got != tt.expected {
				t.Errorf("getStateCategory(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

// TestGetTabHint tests the getTabHint method
func TestGetTabHint(t *testing.T) {
	tests := []struct {
		name         string
		mode         appMode
		tab          sprintTab
		backlogTab   backlogTab
		setupSprints func(m *model)
		expectedText string
	}{
		{
			name: "Sprint mode with dates",
			mode: sprintMode,
			tab:  currentSprint,
			setupSprints: func(m *model) {
				m.sprints[currentSprint] = createTestSprint("Sprint 1", "Project\\Sprint 1", "2024-01-01", "2024-01-14")
			},
			expectedText: "Sprint: 2024-01-01 to 2024-01-14",
		},
		{
			name: "Sprint mode without sprint data",
			mode: sprintMode,
			tab:  currentSprint,
			setupSprints: func(m *model) {
				m.sprints[currentSprint] = nil
			},
			expectedText: "",
		},
		{
			name:         "Backlog mode - recent backlog",
			mode:         backlogMode,
			backlogTab:   recentBacklog,
			setupSprints: func(m *model) {},
			expectedText: "Items not in any sprint, created or updated in the last 30 days",
		},
		{
			name:         "Backlog mode - abandoned work",
			mode:         backlogMode,
			backlogTab:   abandonedWork,
			setupSprints: func(m *model) {},
			expectedText: "Items not updated in the last 14 days (excluding current sprint)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				currentMode:       tt.mode,
				currentTab:        tt.tab,
				currentBacklogTab: tt.backlogTab,
				sprints:           make(map[sprintTab]*Sprint),
			}

			tt.setupSprints(&m)

			got := m.getTabHint()

			if !strings.Contains(got, tt.expectedText) && tt.expectedText != "" {
				t.Errorf("getTabHint() = %q, want to contain %q", got, tt.expectedText)
			}

			if tt.expectedText == "" && got != "" {
				t.Errorf("getTabHint() = %q, want empty string", got)
			}
		})
	}
}

// TestBuildDetailContent tests the buildDetailContent method
func TestBuildDetailContent(t *testing.T) {
	tests := []struct {
		name         string
		task         *WorkItem
		parentTask   *WorkItem
		expectedNil  bool
		checkContent func(content string) bool
	}{
		{
			name:        "Nil task",
			task:        nil,
			expectedNil: true,
		},
		{
			name: "Task with all fields",
			task: &WorkItem{
				ID:           1,
				Title:        "Test Task",
				State:        "Active",
				AssignedTo:   "John Doe",
				WorkItemType: "Task",
				Description:  "This is a test description",
				Tags:         "tag1; tag2",
				Priority:     1,
			},
			expectedNil: false,
			checkContent: func(content string) bool {
				return strings.Contains(content, "Test Task") &&
					strings.Contains(content, "Active") &&
					strings.Contains(content, "John Doe")
			},
		},
		{
			name: "Task with parent",
			task: &WorkItem{
				ID:           2,
				Title:        "Child Task",
				State:        "Active",
				WorkItemType: "Task",
				ParentID:     intPtr(1),
			},
			parentTask: &WorkItem{
				ID:    1,
				Title: "Parent Task",
			},
			expectedNil: false,
			checkContent: func(content string) bool {
				return strings.Contains(content, "Child Task") &&
					strings.Contains(content, "Parent")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sprintLists:  make(map[sprintTab]*WorkItemList),
				backlogLists: make(map[backlogTab]*WorkItemList),
				currentMode:  sprintMode,
				currentTab:   currentSprint,
				selectedTask: tt.task,
				styles:       NewStyles(),
				ui: UIState{
					width: 100,
				},
			}

			// Setup parent task if needed
			if tt.parentTask != nil {
				tasks := []WorkItem{*tt.parentTask, *tt.task}
				m.sprintLists[currentSprint] = createTestList(tasks)
			}

			got := m.buildDetailContent()

			if tt.expectedNil && got != "" {
				t.Error("Expected empty string, got non-empty content")
			}

			if !tt.expectedNil && got == "" {
				t.Error("Expected non-empty content, got empty string")
			}

			if tt.checkContent != nil && !tt.checkContent(got) {
				t.Error("Content check failed")
			}
		})
	}
}
