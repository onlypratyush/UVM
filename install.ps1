<#
.SYNOPSIS
    uvm (Universal Version Manager) - Windows Installer
.DESCRIPTION
    Installs the uvm binary to $HOME\.uvm\bin and adds it to the User PATH environment variable.
.EXAMPLE
    # Online one-liner:
    irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex

    # Local execution:
    powershell -ExecutionPolicy Bypass -File .\install.ps1

    # Uninstallation:
    powershell -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
#>

[CmdletBinding()]
param (
    [Parameter(Mandatory = $false)]
    [string]$InstallDir = "$HOME\.uvm\bin",

    [Parameter(Mandatory = $false)]
    [string]$Version = "latest",

    [Parameter(Mandatory = $false)]
    [string]$Repo = "onlypratyush/UVM-",

    [Parameter(Mandatory = $false)]
    [switch]$Uninstall,

    [Parameter(Mandatory = $false)]
    [switch]$NoPathUpdate
)

$ErrorActionPreference = 'Stop'

function Write-Banner {
    Write-Host ""
    Write-Host "  ██╗   ██╗██╗   ██╗███╗   ███╗" -ForegroundColor Cyan
    Write-Host "  ██║   ██║██║   ██║████╗ ████║" -ForegroundColor Cyan
    Write-Host "  ██║   ██║██║   ██║██╔████╔██║" -ForegroundColor Cyan
    Write-Host "  ██║   ██║╚██╗ ██╔╝██║╚██╔╝██║" -ForegroundColor Cyan
    Write-Host "  ╚██████╔╝ ╚████╔╝ ██║ ╚═╝ ██║" -ForegroundColor Cyan
    Write-Host "   ╚═════╝   ╚═══╝  ╚═╝     ╚═╝" -ForegroundColor Cyan
    Write-Host "  Universal Version Manager for Windows" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Detect-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86"   { return "386" }
        Default { return "amd64" }
    }
}

function Update-UserPath {
    param([string]$TargetDir)

    if ($NoPathUpdate) {
        Write-Info "Skipping PATH update (-NoPathUpdate specified)."
        return
    }

    try {
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        $pathList = $userPath -split ';' | Where-Object { $_ -ne "" }

        if ($pathList -notcontains $TargetDir) {
            $newPath = ($pathList + $TargetDir) -join ';'
            [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
            Write-Success "Added $TargetDir to User PATH environment variable."
        } else {
            Write-Info "$TargetDir is already in User PATH."
        }

        # Update current session PATH as well
        $sessionPaths = $env:Path -split ';'
        if ($sessionPaths -notcontains $TargetDir) {
            $env:Path = "$TargetDir;$env:Path"
        }
    } catch {
        Write-Warn "Could not update User PATH automatically: $_"
        Write-Warn "Please manually add '$TargetDir' to your PATH."
    }
}

function Uninstall-Uvm {
    Write-Banner
    Write-Info "Uninstalling uvm for Windows..."

    $exePath = Join-Path $InstallDir "uvm.exe"
    if (Test-Path $exePath) {
        Remove-Item -Path $exePath -Force -ErrorAction SilentlyContinue
        Write-Success "Removed binary: $exePath"
    } else {
        Write-Warn "Binary not found at $exePath"
    }

    $parentDir = Split-Path $InstallDir -Parent
    if (Test-Path $InstallDir) {
        $remaining = Get-ChildItem -Path $InstallDir -ErrorAction SilentlyContinue
        if ($remaining.Count -eq 0) {
            Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
            Write-Success "Removed empty directory: $InstallDir"
        }
    }

    # Remove from User PATH
    try {
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        $pathList = $userPath -split ';' | Where-Object { $_ -ne "" -and $_ -ne $InstallDir }
        $cleanedPath = $pathList -join ';'
        [Environment]::SetEnvironmentVariable("Path", $cleanedPath, "User")
        Write-Success "Removed $InstallDir from User PATH."
    } catch {
        Write-Warn "Could not clean User PATH automatically: $_"
    }

    Write-Success "uvm has been successfully uninstalled from Windows!"
}

function Install-Uvm {
    Write-Banner
    $arch = Detect-Arch
    Write-Info "Target Platform: windows/$arch"
    Write-Info "Installation Directory: $InstallDir"

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $binPath = Join-Path $InstallDir "uvm.exe"
    $installed = $false

    # 1. Check local directory if script is run locally inside repository
    $scriptDir = $PSScriptRoot
    if ($scriptDir) {
        $localExe = Join-Path $scriptDir "uvm.exe"
        $localMainGo = Join-Path $scriptDir "main.go"

        if (Test-Path $localExe) {
            Write-Info "Found local binary at $localExe, copying..."
            Copy-Item -Path $localExe -Destination $binPath -Force
            $installed = $true
        } elseif ((Test-Path $localMainGo) -and (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-Info "Building uvm from source using local Go..."
            Push-Location $scriptDir
            try {
                go build -ldflags="-s -w" -o $binPath main.go
                $installed = $true
            } finally {
                Pop-Location
            }
        }
    }

    # 2. Download from GitHub Releases
    if (-not $installed) {
        $targetName = "uvm-windows-$arch.exe"
        if ($Version -eq "latest") {
            $downloadUrl = "https://github.com/$Repo/releases/latest/download/$targetName"
        } else {
            $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$targetName"
        }

        Write-Info "Downloading $targetName from GitHub Releases ($downloadUrl)..."
        $tmpFile = [System.IO.Path]::GetTempFileName() + ".exe"

        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            Invoke-WebRequest -Uri $downloadUrl -OutFile $tmpFile -UseBasicParsing
            Move-Item -Path $tmpFile -Destination $binPath -Force
            $installed = $true
        } catch {
            if (Test-Path $tmpFile) { Remove-Item $tmpFile -Force -ErrorAction SilentlyContinue }

            # Fallback to go install if Go is present
            if (Get-Command go -ErrorAction SilentlyContinue) {
                Write-Warn "Release binary download failed. Falling back to Go install..."
                $env:GOBIN = $InstallDir
                go install "github.com/$Repo@latest" 2>$null
                if (Test-Path $binPath) {
                    $installed = $true
                }
            }

            if (-not $installed) {
                Write-Err "Failed to download or build uvm.exe."
                Write-Err "Ensure internet access is available, or install Go (https://go.dev) to build from source."
                exit 1
            }
        }
    }

    Update-UserPath -TargetDir $InstallDir

    Write-Host ""
    Write-Success "uvm was installed successfully to $binPath!"
    Write-Host ""
    Write-Host "Quick Start:" -ForegroundColor Cyan
    Write-Host "  1. Open a new PowerShell / Command Prompt window."
    Write-Host "  2. Verify installation: uvm --help"
    Write-Host "  3. Install a runtime:   uvm install node 20.11.0"
    Write-Host ""
}

if ($Uninstall) {
    Uninstall-Uvm
} else {
    Install-Uvm
}
