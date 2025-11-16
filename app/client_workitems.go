package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// =============================================================================
// WORK ITEM CRUD OPERATIONS
// =============================================================================

// GetWorkItemByID fetches a single work item by its ID
func (c *AzureDevOpsClient) GetWorkItemByID(id int) (*WorkItem, error) {
	ids := []int{id}
	workItemsArgs := workitemtracking.GetWorkItemsArgs{
		Ids:    &ids,
		Expand: &workitemtracking.WorkItemExpandValues.All,
	}

	workItems, err := c.workItemClient.GetWorkItems(c.ctx, workItemsArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item: %w", err)
	}

	if workItems == nil || len(*workItems) == 0 {
		return nil, fmt.Errorf("work item not found")
	}

	task := c.convertWorkItem((*workItems)[0])
	return &task, nil
}

// GetWorkItems returns work items using default parameters
func (c *AzureDevOpsClient) GetWorkItems() ([]WorkItem, error) {
	return c.GetWorkItemsExcluding(nil, "", 40)
}

// GetWorkItemsForSprint returns work items for a specific sprint
func (c *AzureDevOpsClient) GetWorkItemsForSprint(sprintPath string, excludeIDs []int, limit int) ([]WorkItem, error) {
	return c.GetWorkItemsExcluding(excludeIDs, sprintPath, limit)
}

// GetWorkItemsExcluding queries work items with optional exclusions and sprint filter
func (c *AzureDevOpsClient) GetWorkItemsExcluding(excludeIDs []int, sprintPath string, limit int) ([]WorkItem, error) {
	// Cap limit at 30
	if limit > 30 {
		limit = 30
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'`, c.project)

	// Add sprint filter if provided
	if sprintPath != "" {
		query += fmt.Sprintf("\nAND [System.IterationPath] = '%s'", sprintPath)
	}

	// Add exclusion clause if there are IDs to exclude
	if len(excludeIDs) > 0 {
		// Convert IDs to string
		var idStrs []string
		for _, id := range excludeIDs {
			idStrs = append(idStrs, fmt.Sprintf("%d", id))
		}
		query += fmt.Sprintf("\nAND [System.Id] NOT IN (%s)", strings.Join(idStrs, ","))
	}

	query += "\nORDER BY [System.ChangedDate] DESC"

	wiql := workitemtracking.Wiql{
		Query: strPtr(query),
	}

	queryArgs := workitemtracking.QueryByWiqlArgs{
		Wiql: &wiql,
		Top:  &limit,
	}

	result, err := c.workItemClient.QueryByWiql(c.ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to query work items: %w", err)
	}

	if result.WorkItems == nil || len(*result.WorkItems) == 0 {
		return []WorkItem{}, nil
	}

	// Extract work item IDs
	var ids []int
	for _, ref := range *result.WorkItems {
		if ref.Id != nil {
			ids = append(ids, *ref.Id)
		}
	}

	if len(ids) == 0 {
		return []WorkItem{}, nil
	}

	// Get full work item details
	workItemsArgs := workitemtracking.GetWorkItemsArgs{
		Ids:    &ids,
		Expand: &workitemtracking.WorkItemExpandValues.All,
	}

	workItems, err := c.workItemClient.GetWorkItems(c.ctx, workItemsArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item details: %w", err)
	}

	// Convert to our WorkItem struct
	var tasks []WorkItem
	for _, wi := range *workItems {
		task := c.convertWorkItem(wi)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetWorkItemsCount returns total count of work items
func (c *AzureDevOpsClient) GetWorkItemsCount() (int, error) {
	return c.GetWorkItemsCountForSprint("")
}

// GetWorkItemsCountForSprint returns count of work items for a specific sprint
func (c *AzureDevOpsClient) GetWorkItemsCountForSprint(sprintPath string) (int, error) {
	// Query for total count of work items
	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'`, c.project)

	// Add sprint filter if provided
	if sprintPath != "" {
		query += fmt.Sprintf("\nAND [System.IterationPath] = '%s'", sprintPath)
	}

	wiql := workitemtracking.Wiql{
		Query: strPtr(query),
	}

	queryArgs := workitemtracking.QueryByWiqlArgs{
		Wiql: &wiql,
	}

	result, err := c.workItemClient.QueryByWiql(c.ctx, queryArgs)
	if err != nil {
		return 0, fmt.Errorf("failed to query work items count: %w", err)
	}

	if result.WorkItems == nil {
		return 0, nil
	}

	return len(*result.WorkItems), nil
}

// UpdateWorkItemState updates the state of a work item
func (c *AzureDevOpsClient) UpdateWorkItemState(workItemID int, newState string) error {
	// Create a patch document to update the state
	op := webapi.OperationValues.Add
	path := "/fields/System.State"

	patchDocument := []webapi.JsonPatchOperation{
		{
			Op:    &op,
			Path:  &path,
			Value: newState,
		},
	}

	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:       &workItemID,
		Document: &patchDocument,
	}

	_, err := c.workItemClient.UpdateWorkItem(c.ctx, updateArgs)
	if err != nil {
		return fmt.Errorf("failed to update work item state: %w", err)
	}

	return nil
}

// UpdateWorkItem updates multiple fields of a work item
func (c *AzureDevOpsClient) UpdateWorkItem(workItemID int, updates map[string]interface{}) error {
	// Create a patch document with multiple operations
	op := webapi.OperationValues.Add
	var patchDocument []webapi.JsonPatchOperation

	// Map of field keys to their Azure DevOps field paths
	fieldMap := map[string]string{
		"title":       "/fields/System.Title",
		"description": "/fields/System.Description",
		"tags":        "/fields/System.Tags",
		"priority":    "/fields/Microsoft.VSTS.Common.Priority",
		"state":       "/fields/System.State",
	}

	// Build patch operations for each field
	for key, value := range updates {
		if fieldPath, ok := fieldMap[key]; ok {
			path := fieldPath
			patchDocument = append(patchDocument, webapi.JsonPatchOperation{
				Op:    &op,
				Path:  &path,
				Value: value,
			})
		}
	}

	if len(patchDocument) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:       &workItemID,
		Document: &patchDocument,
	}

	_, err := c.workItemClient.UpdateWorkItem(c.ctx, updateArgs)
	if err != nil {
		return fmt.Errorf("failed to update work item: %w", err)
	}

	return nil
}

// DeleteWorkItem deletes a work item by ID
func (c *AzureDevOpsClient) DeleteWorkItem(workItemID int) error {
	deleteArgs := workitemtracking.DeleteWorkItemArgs{
		Id: &workItemID,
	}

	_, err := c.workItemClient.DeleteWorkItem(c.ctx, deleteArgs)
	if err != nil {
		return fmt.Errorf("failed to delete work item: %w", err)
	}

	return nil
}

// CreateWorkItem creates a new work item in Azure DevOps
func (c *AzureDevOpsClient) CreateWorkItem(title string, workItemType string, iterationPath string, parentID *int, areaPath string) (*WorkItem, error) {
	// Build patch document
	op := webapi.OperationValues.Add
	var patchDoc []webapi.JsonPatchOperation

	// Required field: Title
	titlePath := "/fields/System.Title"
	patchDoc = append(patchDoc, webapi.JsonPatchOperation{
		Op:    &op,
		Path:  &titlePath,
		Value: title,
	})

	// Required field: AreaPath
	areaPathField := "/fields/System.AreaPath"
	if areaPath == "" {
		areaPath = c.project // Default to project root if not provided
	}
	patchDoc = append(patchDoc, webapi.JsonPatchOperation{
		Op:    &op,
		Path:  &areaPathField,
		Value: areaPath,
	})

	// Add iteration path if provided, otherwise default to project root
	iterPathField := "/fields/System.IterationPath"
	if iterationPath != "" {
		patchDoc = append(patchDoc, webapi.JsonPatchOperation{
			Op:    &op,
			Path:  &iterPathField,
			Value: iterationPath,
		})
	} else {
		patchDoc = append(patchDoc, webapi.JsonPatchOperation{
			Op:    &op,
			Path:  &iterPathField,
			Value: c.project,
		})
	}

	// Add parent relationship if provided
	if parentID != nil {
		parentURL := fmt.Sprintf("%s/_apis/wit/workItems/%d", c.organizationURL, *parentID)
		relPath := "/relations/-"
		patchDoc = append(patchDoc, webapi.JsonPatchOperation{
			Op:   &op,
			Path: &relPath,
			Value: map[string]interface{}{
				"rel": "System.LinkTypes.Hierarchy-Reverse",
				"url": parentURL,
			},
		})
	}

	// Create work item
	args := workitemtracking.CreateWorkItemArgs{
		Document: &patchDoc,
		Project:  &c.project,
		Type:     &workItemType,
	}

	createdItem, err := c.workItemClient.CreateWorkItem(c.ctx, args)
	if err != nil {
		// Provide detailed error information for debugging
		return nil, fmt.Errorf("failed to create work item (Type: %s, Project: %s, IterationPath: %s, HasParent: %v): %w",
			workItemType, c.project, iterationPath, parentID != nil, err)
	}

	// Convert to WorkItem and return
	workItem := c.convertWorkItem(*createdItem)
	return &workItem, nil
}

// GetWorkItemTypeStates returns valid states for a work item type
func (c *AzureDevOpsClient) GetWorkItemTypeStates(workItemType string) ([]string, map[string]string, error) {
	// Get the work item type definition to get valid states with categories
	getTypeArgs := workitemtracking.GetWorkItemTypeArgs{
		Project: &c.project,
		Type:    &workItemType,
	}

	workItemTypeDef, err := c.workItemClient.GetWorkItemType(c.ctx, getTypeArgs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get work item type: %w", err)
	}

	// Extract states and their categories
	var states []string
	stateCategories := make(map[string]string)

	if workItemTypeDef.States != nil && len(*workItemTypeDef.States) > 0 {
		// Use the States field which includes category information
		for _, state := range *workItemTypeDef.States {
			if state.Name != nil && *state.Name != "" {
				stateName := *state.Name
				states = append(states, stateName)

				// Map state name to its category
				if state.Category != nil {
					stateCategories[stateName] = *state.Category
				} else {
					stateCategories[stateName] = "Unknown"
				}
			}
		}
	} else {
		// Fallback: Extract unique states from transitions if States field is not available
		stateMap := make(map[string]bool)

		if workItemTypeDef.Transitions != nil {
			transitions := *workItemTypeDef.Transitions

			// Add all "from" states (keys in the map)
			for fromState := range transitions {
				if fromState != "" {
					stateMap[fromState] = true
				}
			}

			// Add all "to" states
			for _, transitionList := range transitions {
				if transitionList != nil {
					for _, transition := range transitionList {
						if transition.To != nil && *transition.To != "" {
							stateMap[*transition.To] = true
						}
					}
				}
			}
		}

		// Convert map to slice
		for state := range stateMap {
			states = append(states, state)
			// Guess category based on common state names
			stateCategories[state] = guessStateCategory(state)
		}
	}

	// If no states found, return a default set
	if len(states) == 0 {
		defaultStates := []string{"New", "Active", "Closed"}
		for _, state := range defaultStates {
			stateCategories[state] = guessStateCategory(state)
		}
		return defaultStates, stateCategories, nil
	}

	return states, stateCategories, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// convertWorkItem converts Azure DevOps WorkItem to our WorkItem struct
func (c *AzureDevOpsClient) convertWorkItem(wi workitemtracking.WorkItem) WorkItem {
	fields := *wi.Fields

	task := WorkItem{
		ID: getIntField(wi.Id),
	}

	if title, ok := fields["System.Title"].(string); ok {
		task.Title = title
	}

	if state, ok := fields["System.State"].(string); ok {
		task.State = state
	}

	if workItemType, ok := fields["System.WorkItemType"].(string); ok {
		task.WorkItemType = workItemType
	}

	if assignedTo, ok := fields["System.AssignedTo"].(map[string]interface{}); ok {
		if displayName, ok := assignedTo["displayName"].(string); ok {
			task.AssignedTo = displayName
		}
	}

	if description, ok := fields["System.Description"].(string); ok {
		task.Description = stripHTML(description)
	}

	// History/Comments field
	if history, ok := fields["System.History"].(string); ok {
		task.Comments = stripHTML(history)
	}

	if tags, ok := fields["System.Tags"].(string); ok {
		task.Tags = tags
	}

	if priority, ok := fields["Microsoft.VSTS.Common.Priority"].(float64); ok {
		task.Priority = int(priority)
	}

	if createdDate, ok := fields["System.CreatedDate"].(string); ok {
		task.CreatedDate = formatDate(createdDate)
	}

	if changedDate, ok := fields["System.ChangedDate"].(string); ok {
		task.ChangedDate = formatDate(changedDate)
	}

	if iterationPath, ok := fields["System.IterationPath"].(string); ok {
		task.IterationPath = iterationPath
	}

	if areaPath, ok := fields["System.AreaPath"].(string); ok {
		task.AreaPath = areaPath
	}

	// Extract parent relationship from relations
	if wi.Relations != nil {
		for _, relation := range *wi.Relations {
			if relation.Rel != nil && *relation.Rel == "System.LinkTypes.Hierarchy-Reverse" {
				// This is a parent link
				if relation.Url != nil {
					// Extract parent ID from URL (format: .../workItems/{id})
					parts := strings.Split(*relation.Url, "/")
					if len(parts) > 0 {
						parentIDStr := parts[len(parts)-1]
						if parentID, err := strconv.Atoi(parentIDStr); err == nil {
							task.ParentID = &parentID
						}
					}
				}
				break
			}
		}
	}

	return task
}

// stripHTML removes HTML tags from a string
func stripHTML(html string) string {
	// Simple HTML tag removal - for production, use a proper HTML parser
	result := html
	result = strings.ReplaceAll(result, "<div>", "\n")
	result = strings.ReplaceAll(result, "</div>", "")
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<p>", "\n")
	result = strings.ReplaceAll(result, "</p>", "")

	// Remove all remaining tags
	for strings.Contains(result, "<") && strings.Contains(result, ">") {
		start := strings.Index(result, "<")
		end := strings.Index(result, ">")
		if start < end {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}

	return strings.TrimSpace(result)
}

// formatDate formats Azure DevOps date strings
func formatDate(dateStr string) string {
	// Azure DevOps returns ISO 8601 format
	// Just take the first part for now (date + time without milliseconds)
	if len(dateStr) > 19 {
		return dateStr[:19]
	}
	return dateStr
}

// guessStateCategory attempts to categorize a state based on common naming patterns
func guessStateCategory(state string) string {
	stateLower := strings.ToLower(state)

	// Proposed/Pending states
	if strings.Contains(stateLower, "new") || strings.Contains(stateLower, "proposed") ||
		strings.Contains(stateLower, "backlog") || strings.Contains(stateLower, "to do") {
		return "Proposed"
	}

	// Completed states
	if strings.Contains(stateLower, "closed") || strings.Contains(stateLower, "done") ||
		strings.Contains(stateLower, "completed") {
		return "Completed"
	}

	// Removed states
	if strings.Contains(stateLower, "removed") || strings.Contains(stateLower, "cut") ||
		strings.Contains(stateLower, "cancelled") {
		return "Removed"
	}

	// InProgress states (default for anything active)
	return "InProgress"
}
