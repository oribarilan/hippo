package main

import (
	"fmt"
	"time"
)

// Test fixtures for reusable test data

// Helper builders that extend the existing createTestWorkItem from main_test.go

// createTestWorkItemWithState creates a work item with a specific state
func createTestWorkItemWithState(id int, title string, state string) WorkItem {
	item := createTestWorkItem(id, title, nil)
	item.State = state
	return item
}

// createTestWorkItemWithIteration creates a work item with a specific iteration path
func createTestWorkItemWithIteration(id int, title string, iterationPath string) WorkItem {
	item := createTestWorkItem(id, title, nil)
	item.IterationPath = iterationPath
	return item
}

// createTestWorkItemWithDate creates a work item with specific dates
func createTestWorkItemWithDate(id int, title string, createdDate, changedDate string) WorkItem {
	item := createTestWorkItem(id, title, nil)
	item.CreatedDate = createdDate
	item.ChangedDate = changedDate
	return item
}

// createTestWorkItemComplete creates a work item with all fields populated
func createTestWorkItemComplete(id int, title, state, assignedTo, workItemType, description, tags string, priority int, iterationPath string, parentID *int) WorkItem {
	return WorkItem{
		ID:            id,
		Title:         title,
		State:         state,
		AssignedTo:    assignedTo,
		WorkItemType:  workItemType,
		Description:   description,
		Tags:          tags,
		Priority:      priority,
		CreatedDate:   time.Now().Format("2006-01-02T15:04:05"),
		ChangedDate:   time.Now().Format("2006-01-02T15:04:05"),
		IterationPath: iterationPath,
		AreaPath:      "Project\\Area",
		ParentID:      parentID,
		Children:      nil,
		Comments:      "",
	}
}

// createTestSprint creates a sprint with default values
func createTestSprint(name, path, startDate, endDate string) *Sprint {
	return &Sprint{
		Name:      name,
		Path:      path,
		StartDate: startDate,
		EndDate:   endDate,
	}
}

// Fixture data sets

// simpleTaskSet returns a set of simple tasks without relationships
func simpleTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItem(1, "Task 1", nil),
		createTestWorkItem(2, "Task 2", nil),
		createTestWorkItem(3, "Task 3", nil),
	}
}

// hierarchicalTaskSet returns tasks with parent-child relationships
func hierarchicalTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItem(1, "Parent 1", nil),
		createTestWorkItem(2, "Child 1.1", intPtr(1)),
		createTestWorkItem(3, "Child 1.2", intPtr(1)),
		createTestWorkItem(4, "Parent 2", nil),
		createTestWorkItem(5, "Child 2.1", intPtr(4)),
	}
}

// orphanedTaskSet returns tasks where some children have missing parents
func orphanedTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItem(1, "Root Task", nil),
		createTestWorkItem(2, "Child of Missing Parent", intPtr(999)),
		createTestWorkItem(3, "Child of 1", intPtr(1)),
	}
}

// deepNestedTaskSet returns tasks with multiple levels of nesting
func deepNestedTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItem(1, "Root", nil),
		createTestWorkItem(2, "Level 1", intPtr(1)),
		createTestWorkItem(3, "Level 2", intPtr(2)),
		createTestWorkItem(4, "Level 3", intPtr(3)),
	}
}

// mixedStateTaskSet returns tasks with different states
func mixedStateTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItemWithState(1, "New Task", "New"),
		createTestWorkItemWithState(2, "Active Task", "Active"),
		createTestWorkItemWithState(3, "Done Task", "Done"),
		createTestWorkItemWithState(4, "Closed Task", "Closed"),
	}
}

// sprintTaskSet returns tasks assigned to different sprints
func sprintTaskSet() []WorkItem {
	return []WorkItem{
		createTestWorkItemWithIteration(1, "Sprint 1 Task", "Project\\Sprint 1"),
		createTestWorkItemWithIteration(2, "Sprint 2 Task", "Project\\Sprint 2"),
		createTestWorkItemWithIteration(3, "No Sprint Task", ""),
	}
}

// dateTaskSet returns tasks with various date patterns for testing date functions
func dateTaskSet() []WorkItem {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02T15:04:05")
	lastWeek := now.AddDate(0, 0, -7).Format("2006-01-02T15:04:05")
	twoWeeksAgo := now.AddDate(0, 0, -14).Format("2006-01-02T15:04:05")
	threeMonthsAgo := now.AddDate(0, -3, 0).Format("2006-01-02T15:04:05")

	return []WorkItem{
		createTestWorkItemWithDate(1, "Recent Task", now.Format("2006-01-02T15:04:05"), now.Format("2006-01-02T15:04:05")),
		createTestWorkItemWithDate(2, "Yesterday Task", yesterday, yesterday),
		createTestWorkItemWithDate(3, "Last Week Task", lastWeek, lastWeek),
		createTestWorkItemWithDate(4, "Two Weeks Ago Task", twoWeeksAgo, twoWeeksAgo),
		createTestWorkItemWithDate(5, "Old Task", threeMonthsAgo, threeMonthsAgo),
	}
}

// largeTaskSet returns a large set of tasks for performance testing
func largeTaskSet(count int) []WorkItem {
	tasks := make([]WorkItem, count)
	for i := 0; i < count; i++ {
		tasks[i] = createTestWorkItem(i+1, fmt.Sprintf("Task %d", i+1), nil)
	}
	return tasks
}
