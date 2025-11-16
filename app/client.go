package main

import (
	"context"
	"fmt"
	"os"

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

	// Get access token from Azure CLI (implementation in client_auth.go)
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
