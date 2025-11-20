package main

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/work"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// AzureDevOpsClient wraps Azure DevOps API clients and provides methods for work item operations
// Work item operations are in client_workitems.go
// Sprint operations are in client_sprints.go
// Backlog operations are in client_backlog.go
// Authentication logic is in client_auth.go
type AzureDevOpsClient struct {
	connection      *azuredevops.Connection
	workItemClient  workitemtracking.Client
	workClient      work.Client
	ctx             context.Context
	organizationURL string
	project         string
	team            string
}

// NewAzureDevOpsClient creates a new Azure DevOps client with authentication from Azure CLI
func NewAzureDevOpsClient(config *Config) (*AzureDevOpsClient, error) {
	// Use config values
	organizationURL := config.OrganizationURL
	project := config.Project
	team := config.Team

	if organizationURL == "" {
		return nil, fmt.Errorf("organization_url is not set")
	}
	if project == "" {
		return nil, fmt.Errorf("project is not set")
	}

	// If team is not set, default to project name
	if team == "" {
		team = project
	}

	// Get access token from Azure CLI (implementation in client_auth.go)
	accessToken, err := getAzureCliToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure CLI token: %w\nPlease run 'az login' first", err)
	}

	// Validate token is not empty and looks valid
	if len(accessToken) < 20 {
		return nil, fmt.Errorf("received invalid token from Azure CLI (too short)")
	}

	// Create a connection to Azure DevOps using the Azure CLI token
	connection := azuredevops.NewPatConnection(organizationURL, accessToken)

	ctx := context.Background()

	// Create work item tracking client
	workItemClient, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item client: %w\n\nOrganization URL: %s\n\nPlease verify:\n  1. URL format is https://dev.azure.com/your-organization (no trailing slash)\n  2. You have access to this Azure DevOps organization\n  3. Run 'az account show' to verify correct account", err, organizationURL)
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

// Helper functions

// strPtr is a helper to get pointer to string
func strPtr(s string) *string {
	return &s
}

// getIntField is a helper to safely get int from pointer
func getIntField(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}
