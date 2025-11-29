package main

import (
	"fmt"
	"sort"
	"time"
)

// DummyBackend provides an in-memory mock implementation of Backend for development.
// All data is stored in memory and resets when the application restarts.
type DummyBackend struct {
	workItems map[int]*WorkItem // In-memory storage keyed by ID
	nextID    int               // Auto-increment ID for new work items
	sprints   struct {
		previous *Sprint
		current  *Sprint
		next     *Sprint
	}
	project string
}

// Compile-time check that DummyBackend implements Backend
var _ Backend = (*DummyBackend)(nil)

// NewDummyBackend creates a new dummy backend with sample data
func NewDummyBackend() *DummyBackend {
	db := &DummyBackend{
		workItems: make(map[int]*WorkItem),
		nextID:    1000,
		project:   "DemoProject",
	}
	db.initializeSprints()
	db.initializeSampleData()
	return db
}

// initializeSprints creates three sprints based on current date
func (db *DummyBackend) initializeSprints() {
	now := time.Now()

	// Previous sprint: ended last week
	prevStart := now.AddDate(0, 0, -21)
	prevEnd := now.AddDate(0, 0, -8)
	db.sprints.previous = &Sprint{
		Name:      "Sprint 23",
		Path:      db.project + "\\Sprint 23",
		StartDate: prevStart.Format("2006-01-02"),
		EndDate:   prevEnd.Format("2006-01-02"),
	}

	// Current sprint: started last week, ends next week
	currStart := now.AddDate(0, 0, -7)
	currEnd := now.AddDate(0, 0, 7)
	db.sprints.current = &Sprint{
		Name:      "Sprint 24",
		Path:      db.project + "\\Sprint 24",
		StartDate: currStart.Format("2006-01-02"),
		EndDate:   currEnd.Format("2006-01-02"),
	}

	// Next sprint: starts next week
	nextStart := now.AddDate(0, 0, 8)
	nextEnd := now.AddDate(0, 0, 21)
	db.sprints.next = &Sprint{
		Name:      "Sprint 25",
		Path:      db.project + "\\Sprint 25",
		StartDate: nextStart.Format("2006-01-02"),
		EndDate:   nextEnd.Format("2006-01-02"),
	}
}

// initializeSampleData creates sample work items with parent-child relationships
func (db *DummyBackend) initializeSampleData() {
	now := time.Now()

	// Helper to create work items
	createItem := func(title, workItemType, state, iterPath string, parentID *int, daysAgo int) *WorkItem {
		id := db.nextID
		db.nextID++
		changedDate := now.AddDate(0, 0, -daysAgo).Format("2006-01-02T15:04:05")
		createdDate := now.AddDate(0, 0, -daysAgo-5).Format("2006-01-02T15:04:05")

		item := &WorkItem{
			ID:            id,
			Title:         title,
			State:         state,
			WorkItemType:  workItemType,
			AssignedTo:    "Demo User",
			Description:   fmt.Sprintf("Description for %s", title),
			IterationPath: iterPath,
			AreaPath:      db.project,
			ParentID:      parentID,
			CreatedDate:   createdDate,
			ChangedDate:   changedDate,
			Priority:      2,
		}
		db.workItems[id] = item
		return item
	}

	// Previous Sprint items (completed)
	story1 := createItem("Implement user authentication", "User Story", "Closed", db.sprints.previous.Path, nil, 10)
	createItem("Design login UI", "Task", "Closed", db.sprints.previous.Path, &story1.ID, 12)
	createItem("Add OAuth support", "Task", "Closed", db.sprints.previous.Path, &story1.ID, 10)

	// Current Sprint items (mix of states)
	story2 := createItem("Dashboard improvements", "User Story", "Active", db.sprints.current.Path, nil, 5)
	createItem("Add charts widget", "Task", "Active", db.sprints.current.Path, &story2.ID, 3)
	createItem("Implement filters", "Task", "New", db.sprints.current.Path, &story2.ID, 2)
	createItem("Chart rendering issue", "Bug", "Active", db.sprints.current.Path, &story2.ID, 1)

	story3 := createItem("Performance optimization", "User Story", "Active", db.sprints.current.Path, nil, 4)
	createItem("Database query caching", "Task", "Active", db.sprints.current.Path, &story3.ID, 2)
	createItem("Optimize API responses", "Task", "New", db.sprints.current.Path, &story3.ID, 1)

	// Next Sprint items (planned)
	story4 := createItem("Mobile responsive design", "User Story", "New", db.sprints.next.Path, nil, 1)
	createItem("Tablet layout", "Task", "New", db.sprints.next.Path, &story4.ID, 1)
	createItem("Phone layout", "Task", "New", db.sprints.next.Path, &story4.ID, 1)

	// Backlog items (no sprint assigned - use project root as iteration path)
	createItem("Export to PDF feature", "User Story", "New", db.project, nil, 5)
	createItem("Memory leak on refresh", "Bug", "New", db.project, nil, 3)

	// Abandoned items (stale - not updated in 14+ days)
	createItem("Update documentation", "Task", "New", db.project, nil, 30)
	createItem("Refactor legacy module", "Task", "Active", db.project, nil, 20)
}

// =============================================================================
// WORK ITEM CRUD OPERATIONS
// =============================================================================

// GetWorkItemByID fetches a single work item by its ID
func (db *DummyBackend) GetWorkItemByID(id int) (*WorkItem, error) {
	if item, exists := db.workItems[id]; exists {
		return item, nil
	}
	return nil, fmt.Errorf("work item %d not found", id)
}

// GetWorkItems returns all work items using default parameters
func (db *DummyBackend) GetWorkItems() ([]WorkItem, error) {
	return db.GetWorkItemsExcluding(nil, "", 40)
}

// GetWorkItemsForSprint returns work items for a specific sprint
func (db *DummyBackend) GetWorkItemsForSprint(sprintPath string, excludeIDs []int, limit int) ([]WorkItem, error) {
	return db.GetWorkItemsExcluding(excludeIDs, sprintPath, limit)
}

// GetWorkItemsExcluding queries work items with optional exclusions and sprint filter
func (db *DummyBackend) GetWorkItemsExcluding(excludeIDs []int, sprintPath string, limit int) ([]WorkItem, error) {
	excludeSet := make(map[int]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	var result []WorkItem
	for _, item := range db.workItems {
		// Skip excluded IDs
		if excludeSet[item.ID] {
			continue
		}

		// Skip closed/removed items
		if item.State == "Closed" || item.State == "Removed" {
			continue
		}

		// Filter by sprint path if provided
		if sprintPath != "" && item.IterationPath != sprintPath {
			continue
		}

		result = append(result, *item)

		if len(result) >= limit {
			break
		}
	}

	// Sort by changed date (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangedDate > result[j].ChangedDate
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetWorkItemsCount returns total count of non-closed work items
func (db *DummyBackend) GetWorkItemsCount() (int, error) {
	return db.GetWorkItemsCountForSprint("")
}

// GetWorkItemsCountForSprint returns count of work items for a specific sprint
func (db *DummyBackend) GetWorkItemsCountForSprint(sprintPath string) (int, error) {
	count := 0
	for _, item := range db.workItems {
		if item.State == "Closed" || item.State == "Removed" {
			continue
		}
		if sprintPath != "" && item.IterationPath != sprintPath {
			continue
		}
		count++
	}
	return count, nil
}

// UpdateWorkItemState updates the state of a work item
func (db *DummyBackend) UpdateWorkItemState(workItemID int, newState string) error {
	if item, exists := db.workItems[workItemID]; exists {
		item.State = newState
		item.ChangedDate = time.Now().Format("2006-01-02T15:04:05")
		return nil
	}
	return fmt.Errorf("work item %d not found", workItemID)
}

// UpdateWorkItem updates multiple fields of a work item
func (db *DummyBackend) UpdateWorkItem(workItemID int, updates map[string]interface{}) error {
	item, exists := db.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item %d not found", workItemID)
	}

	for key, value := range updates {
		switch key {
		case "title":
			if v, ok := value.(string); ok {
				item.Title = v
			}
		case "description":
			if v, ok := value.(string); ok {
				item.Description = v
			}
		case "state":
			if v, ok := value.(string); ok {
				item.State = v
			}
		case "tags":
			if v, ok := value.(string); ok {
				item.Tags = v
			}
		case "priority":
			if v, ok := value.(int); ok {
				item.Priority = v
			}
		}
	}

	item.ChangedDate = time.Now().Format("2006-01-02T15:04:05")
	return nil
}

// CreateWorkItem creates a new work item
func (db *DummyBackend) CreateWorkItem(title string, workItemType string, iterationPath string, parentID *int, areaPath string) (*WorkItem, error) {
	id := db.nextID
	db.nextID++

	now := time.Now().Format("2006-01-02T15:04:05")

	if iterationPath == "" {
		iterationPath = db.project
	}
	if areaPath == "" {
		areaPath = db.project
	}

	item := &WorkItem{
		ID:            id,
		Title:         title,
		State:         "New",
		WorkItemType:  workItemType,
		AssignedTo:    "Demo User",
		Description:   "",
		IterationPath: iterationPath,
		AreaPath:      areaPath,
		ParentID:      parentID,
		CreatedDate:   now,
		ChangedDate:   now,
		Priority:      2,
	}

	db.workItems[id] = item
	return item, nil
}

// DeleteWorkItem deletes a work item by ID
func (db *DummyBackend) DeleteWorkItem(workItemID int) error {
	if _, exists := db.workItems[workItemID]; exists {
		delete(db.workItems, workItemID)
		return nil
	}
	return fmt.Errorf("work item %d not found", workItemID)
}

// MoveWorkItemToSprint moves a work item to a specific sprint
func (db *DummyBackend) MoveWorkItemToSprint(workItemID int, iterationPath string) error {
	item, exists := db.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item %d not found", workItemID)
	}

	if iterationPath == "" {
		iterationPath = db.project // Move to backlog
	}
	item.IterationPath = iterationPath
	item.ChangedDate = time.Now().Format("2006-01-02T15:04:05")
	return nil
}

// GetWorkItemTypeStates returns valid states for a work item type
func (db *DummyBackend) GetWorkItemTypeStates(workItemType string) ([]string, map[string]string, error) {
	states := []string{"New", "Active", "Closed", "Removed"}
	categories := map[string]string{
		"New":     "Proposed",
		"Active":  "InProgress",
		"Closed":  "Completed",
		"Removed": "Removed",
	}
	return states, categories, nil
}

// =============================================================================
// SPRINT OPERATIONS
// =============================================================================

// GetCurrentAndAdjacentSprints returns previous, current, and next sprint
func (db *DummyBackend) GetCurrentAndAdjacentSprints() (prev *Sprint, curr *Sprint, next *Sprint, err error) {
	return db.sprints.previous, db.sprints.current, db.sprints.next, nil
}

// =============================================================================
// BACKLOG OPERATIONS
// =============================================================================

// GetRecentBacklogItems returns work items not assigned to any sprint, recently updated
func (db *DummyBackend) GetRecentBacklogItems(limit int) ([]WorkItem, error) {
	return db.GetRecentBacklogItemsExcluding(nil, limit)
}

// GetRecentBacklogItemsExcluding returns recent backlog items excluding specific IDs
func (db *DummyBackend) GetRecentBacklogItemsExcluding(excludeIDs []int, limit int) ([]WorkItem, error) {
	excludeSet := make(map[int]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var result []WorkItem
	for _, item := range db.workItems {
		if excludeSet[item.ID] {
			continue
		}

		// Must be in backlog (project root iteration path)
		if item.IterationPath != db.project {
			continue
		}

		// Must not be closed/removed
		if item.State == "Closed" || item.State == "Removed" || item.State == "Completed" || item.State == "Done" {
			continue
		}

		// Must be recently updated (within 30 days)
		changedDate, err := time.Parse("2006-01-02T15:04:05", item.ChangedDate)
		if err != nil {
			continue
		}
		if changedDate.Before(thirtyDaysAgo) {
			continue
		}

		result = append(result, *item)
	}

	// Sort by changed date (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangedDate > result[j].ChangedDate
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetRecentBacklogItemsCount returns the count of recent backlog items
func (db *DummyBackend) GetRecentBacklogItemsCount() (int, error) {
	items, err := db.GetRecentBacklogItemsExcluding(nil, 1000)
	return len(items), err
}

// GetAbandonedWorkItems returns work items not updated in 14+ days
func (db *DummyBackend) GetAbandonedWorkItems(currentSprintPath string, limit int) ([]WorkItem, error) {
	return db.GetAbandonedWorkItemsExcluding(nil, currentSprintPath, limit)
}

// GetAbandonedWorkItemsExcluding returns abandoned work items excluding specific IDs
func (db *DummyBackend) GetAbandonedWorkItemsExcluding(excludeIDs []int, currentSprintPath string, limit int) ([]WorkItem, error) {
	excludeSet := make(map[int]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	fourteenDaysAgo := time.Now().AddDate(0, 0, -14)

	var result []WorkItem
	for _, item := range db.workItems {
		if excludeSet[item.ID] {
			continue
		}

		// Must not be in current sprint
		if currentSprintPath != "" && item.IterationPath == currentSprintPath {
			continue
		}

		// Must not be closed/removed
		if item.State == "Closed" || item.State == "Removed" || item.State == "Completed" || item.State == "Done" {
			continue
		}

		// Must be stale (not updated in 14+ days)
		changedDate, err := time.Parse("2006-01-02T15:04:05", item.ChangedDate)
		if err != nil {
			continue
		}
		if changedDate.After(fourteenDaysAgo) {
			continue
		}

		result = append(result, *item)
	}

	// Sort by changed date (oldest first for abandoned items)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangedDate < result[j].ChangedDate
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetAbandonedWorkItemsCount returns the count of abandoned work items
func (db *DummyBackend) GetAbandonedWorkItemsCount(currentSprintPath string) (int, error) {
	items, err := db.GetAbandonedWorkItemsExcluding(nil, currentSprintPath, 1000)
	return len(items), err
}
