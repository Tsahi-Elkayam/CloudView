# CloudView PowerShell Build Script
# Usage: .\build.ps1 [command]

param(
    [string]$Command = "help"
)

# Variables
$BinaryName = "cloudview.exe"
$Version = if ($env:VERSION) { $env:VERSION } else { "dev" }
$BuildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
$GitCommit = try { git rev-parse HEAD 2>$null } catch { "unknown" }
$BuildDir = "build"
$DistDir = "dist"

# Build flags
$LdFlags = "-ldflags `"-X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.version=$Version -X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.buildTime=$BuildTime -X github.com/Tsahi-Elkayam/cloudview/cmd/cloudview.gitCommit=$GitCommit`""

function Show-Help {
    Write-Host "CloudView PowerShell Build Script" -ForegroundColor Green
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [command]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Available commands:" -ForegroundColor Yellow
    Write-Host "  deps         Download dependencies"
    Write-Host "  build        Build the binary"
    Write-Host "  clean        Clean build artifacts"
    Write-Host "  test         Run tests"
    Write-Host "  fmt          Format code"
    Write-Host "  vet          Vet code"
    Write-Host "  run          Build and run"
    Write-Host "  dev          Run in development mode"
    Write-Host "  help         Show this help"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  .\build.ps1 deps"
    Write-Host "  .\build.ps1 build"
    Write-Host "  .\build.ps1 run"
}

function Get-Dependencies {
    Write-Host "Downloading dependencies..." -ForegroundColor Green
    go mod download
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    go mod tidy
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    
    # Install testify for testing if not present
    Write-Host "Ensuring test dependencies..." -ForegroundColor Yellow
    go get github.com/stretchr/testify/assert 2>$null
    go get github.com/stretchr/testify/require 2>$null
    
    Write-Host "Dependencies downloaded successfully!" -ForegroundColor Green
}

function Build-Binary {
    Write-Host "Building $BinaryName..." -ForegroundColor Green
    
    # Create build directory
    if (!(Test-Path $BuildDir)) {
        New-Item -ItemType Directory -Path $BuildDir | Out-Null
    }
    
    # Build the binary
    $BuildCmd = "go build $LdFlags -o $BuildDir/$BinaryName ./cmd/main.go"
    Write-Host "Executing: $BuildCmd" -ForegroundColor Yellow
    Invoke-Expression $BuildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build completed successfully!" -ForegroundColor Green
        Write-Host "Binary location: $BuildDir/$BinaryName" -ForegroundColor Cyan
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

function Clean-Artifacts {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Green
    
    if (Test-Path $BuildDir) {
        Remove-Item -Recurse -Force $BuildDir
        Write-Host "Removed $BuildDir directory" -ForegroundColor Yellow
    }
    
    if (Test-Path $DistDir) {
        Remove-Item -Recurse -Force $DistDir
        Write-Host "Removed $DistDir directory" -ForegroundColor Yellow
    }
    
    # Clean Go cache
    go clean
    Write-Host "Cleaned successfully!" -ForegroundColor Green
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Green
    go test -v -race -coverprofile=coverage.out ./...
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "All tests passed!" -ForegroundColor Green
    } else {
        Write-Host "Some tests failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

function Format-Code {
    Write-Host "Formatting code..." -ForegroundColor Green
    go fmt ./...
    Write-Host "Code formatted successfully!" -ForegroundColor Green
}

function Vet-Code {
    Write-Host "Vetting code..." -ForegroundColor Green
    go vet ./...
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Code vetting passed!" -ForegroundColor Green
    } else {
        Write-Host "Code vetting failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

function Run-Application {
    Build-Binary
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Running $BinaryName..." -ForegroundColor Green
        & ".\$BuildDir\$BinaryName"
    }
}

function Run-Dev {
    Write-Host "Running in development mode..." -ForegroundColor Green
    go run ./cmd/main.go
}

function Build-All-Platforms {
    Write-Host "Building for multiple platforms..." -ForegroundColor Green
    
    # Create dist directory
    if (!(Test-Path $DistDir)) {
        New-Item -ItemType Directory -Path $DistDir | Out-Null
    }
    
    # Define platforms
    $platforms = @(
        @{GOOS="linux"; GOARCH="amd64"; EXT=""},
        @{GOOS="linux"; GOARCH="arm64"; EXT=""},
        @{GOOS="darwin"; GOARCH="amd64"; EXT=""},
        @{GOOS="darwin"; GOARCH="arm64"; EXT=""},
        @{GOOS="windows"; GOARCH="amd64"; EXT=".exe"}
    )
    
    foreach ($platform in $platforms) {
        $outputName = "$DistDir/cloudview-$($platform.GOOS)-$($platform.GOARCH)$($platform.EXT)"
        Write-Host "Building for $($platform.GOOS)/$($platform.GOARCH)..." -ForegroundColor Yellow
        
        $env:GOOS = $platform.GOOS
        $env:GOARCH = $platform.GOARCH
        
        $buildCmd = "go build $LdFlags -o $outputName ./cmd/main.go"
        Invoke-Expression $buildCmd
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ $outputName" -ForegroundColor Green
        } else {
            Write-Host "✗ Failed to build $outputName" -ForegroundColor Red
        }
    }
    
    # Reset environment
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
}

# Main script logic
switch ($Command.ToLower()) {
    "deps" { Get-Dependencies }
    "build" { Build-Binary }
    "clean" { Clean-Artifacts }
    "test" { Run-Tests }
    "fmt" { Format-Code }
    "vet" { Vet-Code }
    "run" { Run-Application }
    "dev" { Run-Dev }
    "build-all" { Build-All-Platforms }
    "all" { 
        Get-Dependencies
        Format-Code
        Vet-Code
        Run-Tests
        Build-Binary
    }
    "help" { Show-Help }
    default { 
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host ""
        Show-Help
        exit 1
    }
}