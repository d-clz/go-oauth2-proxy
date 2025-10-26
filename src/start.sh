#!/bin/bash

# Token Gateway - Quick Start Script
# This script helps you get started with the Token Gateway locally

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}OAuth2 Token Gateway - Quick Start${NC}"
echo -e "${GREEN}================================${NC}"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go 1.21+ first"
    echo "Visit: https://go.dev/dl/"
    exit 1
fi

echo -e "${GREEN}✓${NC} Go is installed: $(go version)"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -f "cmd/gateway/main.go" ]; then
    echo -e "${RED}Error: Please run this script from the go-oauth2-proxy directory${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} In correct directory"

# Download dependencies
echo ""
echo "Downloading dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓${NC} Dependencies downloaded"

# Check for service account key
echo ""
if [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo -e "${YELLOW}⚠${NC}  GOOGLE_APPLICATION_CREDENTIALS not set"
    echo ""
    read -p "Enter path to your service account JSON key: " KEY_PATH
    
    if [ ! -f "$KEY_PATH" ]; then
        echo -e "${RED}Error: File not found: $KEY_PATH${NC}"
        exit 1
    fi
    
    export GOOGLE_APPLICATION_CREDENTIALS="$KEY_PATH"
    echo -e "${GREEN}✓${NC} Using credentials: $KEY_PATH"
else
    echo -e "${GREEN}✓${NC} Using credentials: $GOOGLE_APPLICATION_CREDENTIALS"
fi

# Check config file
echo ""
if [ ! -f "config.yaml" ]; then
    echo -e "${RED}Error: config.yaml not found${NC}"
    echo "Please create config.yaml first"
    exit 1
fi

echo -e "${GREEN}✓${NC} Config file found"

# Display config summary
echo ""
echo "Configuration Summary:"
echo "----------------------"
UPSTREAMS=$(grep -c "name:" config.yaml || echo "0")
echo "  Upstreams configured: $UPSTREAMS"
echo "  Server will listen on: $(grep -A2 'server:' config.yaml | grep 'address:' | awk '{print $2}'):$(grep -A3 'server:' config.yaml | grep 'port:' | awk '{print $2}')"
echo ""

# Ask if user wants to build or run directly
echo "What would you like to do?"
echo "  1) Run directly (go run)"
echo "  2) Build binary first"
echo "  3) Run with debug logging"
read -p "Choose (1-3): " CHOICE

echo ""
case $CHOICE in
    1)
        echo "Starting Token Gateway..."
        echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
        echo ""
        go run cmd/gateway/main.go -config config.yaml
        ;;
    2)
        echo "Building binary..."
        go build -o go-oauth2-proxy cmd/gateway/main.go
        echo -e "${GREEN}✓${NC} Built: ./go-oauth2-proxy"
        echo ""
        echo "Starting Token Gateway..."
        echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
        echo ""
        ./go-oauth2-proxy -config config.yaml
        ;;
    3)
        echo "Starting Token Gateway with DEBUG logging..."
        echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
        echo ""
        go run cmd/gateway/main.go -config config.yaml -log-level debug
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac
