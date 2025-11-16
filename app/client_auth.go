package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// getAzureCliToken retrieves an access token from Azure CLI
func getAzureCliToken() (string, error) {
	// Use Azure CLI to get an access token for Azure DevOps with a 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "az", "account", "get-access-token", "--resource", "499b84ac-1321-427f-aa17-267ca6975798", "--query", "accessToken", "-o", "tsv")

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("azure cli token acquisition timed out after 10 seconds")
		}
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
