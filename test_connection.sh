#!/bin/bash

# Simple connection test script for Hippo

echo "===== Hippo Azure DevOps Connection Test ====="
echo ""

# Check if .env exists
if [ ! -f "app/.env" ]; then
    echo "❌ Error: app/.env file not found"
    echo "   Please create it from app/.env.example"
    exit 1
fi

# Load and display env vars (masked)
echo "📋 Configuration:"
source app/.env 2>/dev/null
echo "   AZURE_DEVOPS_ORG_URL: ${AZURE_DEVOPS_ORG_URL}"
echo "   AZURE_DEVOPS_PROJECT: ${AZURE_DEVOPS_PROJECT}"
echo "   AZURE_DEVOPS_TEAM: ${AZURE_DEVOPS_TEAM}"
echo ""

# Check Azure CLI
echo "🔍 Checking Azure CLI..."
if ! command -v az &> /dev/null; then
    echo "❌ Azure CLI not found. Please install it first."
    exit 1
fi
echo "✅ Azure CLI found"
echo ""

# Check if logged in
echo "🔐 Checking Azure login status..."
if ! az account show &> /dev/null; then
    echo "❌ Not logged in to Azure"
    echo "   Please run: az login"
    exit 1
fi

ACCOUNT_NAME=$(az account show --query name -o tsv)
echo "✅ Logged in as: $ACCOUNT_NAME"
echo ""

# Test getting token
echo "🎫 Testing token acquisition..."
TOKEN=$(az account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken -o tsv 2>&1)
if [ $? -ne 0 ]; then
    echo "❌ Failed to get token"
    echo "   Error: $TOKEN"
    exit 1
fi

TOKEN_LEN=${#TOKEN}
echo "✅ Token acquired successfully (length: $TOKEN_LEN chars)"
echo ""

# Validate URL format
if [[ ! "$AZURE_DEVOPS_ORG_URL" =~ ^https://dev\.azure\.com/[^/]+$ ]]; then
    echo "⚠️  Warning: Organization URL format looks suspicious"
    echo "   Expected: https://dev.azure.com/your-organization"
    echo "   Got:      $AZURE_DEVOPS_ORG_URL"
    echo ""
    echo "   Common issues:"
    echo "   - Trailing slash: https://dev.azure.com/org/ ❌"
    echo "   - Project in URL: https://dev.azure.com/org/project ❌"
    echo "   - Correct format: https://dev.azure.com/org ✅"
    echo ""
fi

echo "===== Connection Test Complete ====="
echo ""
echo "If you still see errors, please verify:"
echo "  1. You have access to the Azure DevOps organization"
echo "  2. The project name is correct and you have access to it"
echo "  3. Your account has work item read/write permissions"
echo ""
echo "To run Hippo: cd app && go run ."
