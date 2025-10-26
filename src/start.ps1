# Token Gateway - Quick Start Script (PowerShell)
$ErrorActionPreference = "Stop"

$RED = "`e[31m"
$GREEN = "`e[32m"
$YELLOW = "`e[33m"
$NC = "`e[0m"

Write-Host "${GREEN}================================${NC}"
Write-Host "${GREEN}OAuth2 Token Gateway - Quick Start${NC}"
Write-Host "${GREEN}================================${NC}"
Write-Host ""

# Check Go installation
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "${RED}Error: Go is not installed${NC}"
    exit 1
}

Write-Host "${GREEN}✓${NC} Go is installed: $(go version)"

# Check correct directory
if (-not (Test-Path "go.mod") -or -not (Test-Path "cmd/gateway/main.go")) {
    Write-Host "${RED}Error: Please run this script from the go-oauth2-proxy directory${NC}"
    exit 1
}

Write-Host "${GREEN}✓${NC} In correct directory"

# Download dependencies
Write-Host ""
Write-Host "Downloading dependencies..."
go mod download
go mod tidy
Write-Host "${GREEN}✓${NC} Dependencies downloaded"

# Check GOOGLE_APPLICATION_CREDENTIALS
Write-Host ""
if (-not $env:GOOGLE_APPLICATION_CREDENTIALS) {
    Write-Host "${YELLOW}⚠${NC} GOOGLE_APPLICATION_CREDENTIALS not set"
    Write-Host ""
    $KEY_PATH = Read-Host "Enter path to your service account JSON key"

    if (-not (Test-Path $KEY_PATH)) {
        Write-Host "${RED}Error: File not found: $KEY_PATH${NC}"
        exit 1
    }

    $env:GOOGLE_APPLICATION_CREDENTIALS = $KEY_PATH
    Write-Host "${GREEN}✓${NC} Using credentials: $KEY_PATH"
} else {
    Write-Host "${GREEN}✓${NC} Using credentials: $env:GOOGLE_APPLICATION_CREDENTIALS"
}

# Check config
Write-Host ""
if (-not (Test-Path "config.yaml")) {
    Write-Host "${RED}Error: config.yaml not found${NC}"
    exit 1
}
Write-Host "${GREEN}✓${NC} Config file found"

# Summary
Write-Host ""
Write-Host "Configuration Summary:"
Write-Host "----------------------"
$upstreams = (Select-String -Path "config.yaml" -Pattern "name:" | Measure-Object).Count
Write-Host "  Upstreams configured: $upstreams"

$address = (Select-String -Path "config.yaml" -Pattern "address:" | ForEach-Object { $_.Line.Split(':')[1].Trim() })
$port = (Select-String -Path "config.yaml" -Pattern "port:" | ForEach-Object { $_.Line.Split(':')[1].Trim() })
Write-Host "  Server will listen on: ${address}:${port}"
Write-Host ""

# Menu
Write-Host "What would you like to do?"
Write-Host "  1) Run directly (go run)"
Write-Host "  2) Build binary first"
Write-Host "  3) Run with debug logging"
$choice = Read-Host "Choose (1-3)"

Write-Host ""

switch ($choice) {
    "1" {
        Write-Host "Starting Token Gateway..."
        Write-Host "${YELLOW}Press Ctrl+C to stop${NC}"
        go run cmd/gateway/main.go -config config.yaml
    }
    "2" {
        Write-Host "Building binary..."
        go build -o go-oauth2-proxy cmd/gateway/main.go
        Write-Host "${GREEN}✓${NC} Built: ./go-oauth2-proxy"
        Write-Host "${YELLOW}Press Ctrl+C to stop${NC}"
        ./go-oauth2-proxy -config config.yaml
    }
    "3" {
        Write-Host "Starting Token Gateway with DEBUG logging..."
        Write-Host "${YELLOW}Press Ctrl+C to stop${NC}"
        go run cmd/gateway/main.go -config config.yaml -log-level debug
    }
    Default {
        Write-Host "${RED}Invalid choice${NC}"
        exit 1
    }
}
