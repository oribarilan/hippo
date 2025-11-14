package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/work"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

type AzureDevOpsClient struct {
	connection      *azuredevops.Connection
	workItemClient  workitemtracking.Client
	workClient      work.Client
	ctx             context.Context
	organizationURL string
	project         string
	team            string
}

func NewAzureDevOpsClient() (*AzureDevOpsClient, error) {
	// Read environment variables
	organizationURL := os.Getenv("AZURE_DEVOPS_ORG_URL")
	project := os.Getenv("AZURE_DEVOPS_PROJECT")
	team := os.Getenv("AZURE_DEVOPS_TEAM")

	if organizationURL == "" {
		return nil, fmt.Errorf("AZURE_DEVOPS_ORG_URL environment variable is not set")
	}
	if project == "" {
		return nil, fmt.Errorf("AZURE_DEVOPS_PROJECT environment variable is not set")
	}

	// If team is not set, default to project name
	if team == "" {
		team = project
	}

	// Get access token from Azure CLI
	accessToken, err := getAzureCliToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure CLI token: %w\nPlease run 'az login' first", err)
	}

	// Create a connection to Azure DevOps using the Azure CLI token
	connection := azuredevops.NewPatConnection(organizationURL, accessToken)

	ctx := context.Background()

	// Create work item tracking client
	workItemClient, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item client: %w", err)
	}

	// Create work client for iterations
	workClient, err := work.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work client: %w", err)
	}

	return &AzureDevOpsClient{
		connection:      connection,
		workItemClient:  workItemClient,
		workClient:      workClient,
		ctx:             ctx,
		organizationURL: organizationURL,
		project:         project,
		team:            team,
	}, nil
}

// getAzureCliToken retrieves an access token from Azure CLI
func getAzureCliToken() (string, error) {
	// Use Azure CLI to get an access token for Azure DevOps
	cmd := exec.Command("az", "account", "get-access-token", "--resource", "499b84ac-1321-427f-aa17-267ca6975798", "--query", "accessToken", "-o", "tsv")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("az cli error: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute az cli: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("received empty token from Azure CLI")
	}

	return token, nil
}

func (c *AzureDevOpsClient) GetWorkItems() ([]WorkItem, error) {
	return c.GetWorkItemsExcluding(nil, "", 30)
}

func (c *AzureDevOpsClient) GetWorkItemsForSprint(sprintPath string, excludeIDs []int, limit int) ([]WorkItem, error) {
	return c.GetWorkItemsExcluding(excludeIDs, sprintPath, limit)
}

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

func (c *AzureDevOpsClient) GetWorkItemsCount() (int, error) {
	return c.GetWorkItemsCountForSprint("")
}

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

func getIntField(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}

func strPtr(s string) *string {
	return &s
}

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

func formatDate(dateStr string) string {
	// Azure DevOps returns ISO 8601 format
	// Just take the first part for now (date + time without milliseconds)
	if len(dateStr) > 19 {
		return dateStr[:19]
	}
	return dateStr
}

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

// GetTeamIterations fetches iterations for the team
func (c *AzureDevOpsClient) GetTeamIterations() ([]work.TeamSettingsIteration, error) {
	iterations, err := c.workClient.GetTeamIterations(c.ctx, work.GetTeamIterationsArgs{
		Project: &c.project,
		Team:    &c.team,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get team iterations: %w", err)
	}

	if iterations == nil {
		return []work.TeamSettingsIteration{}, nil
	}

	return *iterations, nil
}

// GetCurrentAndAdjacentSprints returns previous, current, and next sprint
func (c *AzureDevOpsClient) GetCurrentAndAdjacentSprints() (prev, curr, next *work.TeamSettingsIteration, err error) {
	iterations, err := c.GetTeamIterations()
	if err != nil {
		return nil, nil, nil, err
	}

	if len(iterations) == 0 {
		return nil, nil, nil, nil
	}

	now := time.Now()
	var currentIdx = -1

	// Find current sprint
	for i, iter := range iterations {
		if iter.Attributes == nil {
			continue
		}

		startDate := iter.Attributes.StartDate
		finishDate := iter.Attributes.FinishDate

		if startDate != nil && finishDate != nil {
			start := startDate.Time
			finish := finishDate.Time

			if now.After(start) && now.Before(finish) {
				currentIdx = i
				curr = &iterations[i]
				break
			}
		}
	}

	// If no current sprint found, use the most recent one
	if currentIdx == -1 && len(iterations) > 0 {
		currentIdx = len(iterations) - 1
		curr = &iterations[currentIdx]
	}

	// Get previous sprint
	if currentIdx > 0 {
		prev = &iterations[currentIdx-1]
	}

	// Get next sprint
	if currentIdx >= 0 && currentIdx < len(iterations)-1 {
		next = &iterations[currentIdx+1]
	}

	return prev, curr, next, nil
}
