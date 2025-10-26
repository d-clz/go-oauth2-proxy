# Fix go.sum and rebuild Docker image
# PowerShell script for Windows

Write-Host "🔧 Fixing go.sum..." -ForegroundColor Cyan
Write-Host ""

# Navigate to project root
$projectRoot = "E:\DUONGHT\go-oauth2-proxy\src"
if (Test-Path $projectRoot) {
    Set-Location $projectRoot
    Write-Host "✓ Changed to project directory: $projectRoot" -ForegroundColor Green
} else {
    Write-Host "⚠ Could not find project directory, using current directory" -ForegroundColor Yellow
}

Write-Host ""

# Step 1: Tidy go.mod and regenerate go.sum
Write-Host "Step 1: Regenerating go.sum..." -ForegroundColor Yellow
go mod tidy

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ go mod tidy completed" -ForegroundColor Green
} else {
    Write-Host "✗ go mod tidy failed" -ForegroundColor Red
    exit 1
}

# Step 2: Verify all dependencies
Write-Host ""
Write-Host "Step 2: Verifying dependencies..." -ForegroundColor Yellow
go mod verify

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Dependencies verified" -ForegroundColor Green
} else {
    Write-Host "⚠ Verification had issues, but continuing..." -ForegroundColor Yellow
}

# Step 3: Download all dependencies
Write-Host ""
Write-Host "Step 3: Downloading dependencies..." -ForegroundColor Yellow
go mod download

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Dependencies downloaded" -ForegroundColor Green
} else {
    Write-Host "✗ Download failed" -ForegroundColor Red
}

Write-Host ""
Write-Host "✅ go.sum has been fixed!" -ForegroundColor Green
Write-Host ""

# Step 4: Test build locally
Write-Host "Step 4: Testing local build..." -ForegroundColor Yellow
go build -v ./cmd/gateway

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Local build successful!" -ForegroundColor Green
    Write-Host ""

    # Step 5: Offer to rebuild Docker
    $rebuild = Read-Host "Do you want to rebuild the Docker image now? (y/n)"

    if ($rebuild -eq "y" -or $rebuild -eq "Y") {
        Write-Host "Building Docker image..." -ForegroundColor Yellow
        Set-Location ..
        docker build -t token-gateway:latest -f deployment/Dockerfile ./src

        if ($LASTEXITCODE -eq 0) {
            Write-Host ""
            Write-Host "✅ Docker build successful!" -ForegroundColor Green
            Write-Host ""
            Write-Host "Next steps:" -ForegroundColor Cyan
            Write-Host "1. Commit the updated go.sum:"
            Write-Host "   git add src/go.sum" -ForegroundColor White
            Write-Host "   git commit -m 'fix: update go.sum with missing dependencies'" -ForegroundColor White
            Write-Host "   git push" -ForegroundColor White
            Write-Host ""
            Write-Host "2. Your GitHub Actions workflow will now work!" -ForegroundColor Green
        } else {
            Write-Host ""
            Write-Host "❌ Docker build failed. Check the error above." -ForegroundColor Red
        }
    }
} else {
    Write-Host ""
    Write-Host "❌ Local build failed. Please check the errors above." -ForegroundColor Red
    Write-Host ""
    Write-Host "Common fixes:" -ForegroundColor Yellow
    Write-Host "1. Make sure you're in the correct directory"
    Write-Host "2. Run: go mod tidy"
    Write-Host "3. Run: go get golang.org/x/oauth2@latest"
    Write-Host "4. Run: go get google.golang.org/api/idtoken@latest"
}