<#
.SYNOPSIS
    uvm (Universal Version Manager) - Windows Installer
.DESCRIPTION
    Installs the uvm binary to $HOME\.uvm\bin, detects existing Node.js installations,
    offers migration/cleanup options, and updates the User PATH environment variable.
.EXAMPLE
    # Online one-liner:
    irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex

    # Local execution:
    powershell -ExecutionPolicy Bypass -File .\install.ps1

    # Automated migration:
    powershell -ExecutionPolicy Bypass -File .\install.ps1 -NodeAction move

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
    [ValidateSet("move", "delete", "keep", "")]
    [string]$NodeAction = "",

    [Parameter(Mandatory = $false)]
    [switch]$ConfirmDelete,

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

function Get-UserPath {
    try {
        return [Environment]::GetEnvironmentVariable("Path", "User")
    } catch {
        return ""
    }
}

function Set-UserPath {
    param([string]$NewPath)
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    
    # Broadcast WM_SETTINGCHANGE
    try {
        Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @"
            [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
            public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@ -ErrorAction SilentlyContinue | Out-Null
        $result = [UIntPtr]::Zero
        [Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x001A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
    } catch {
        # Ignore if P/Invoke fails in restricted environments
    }
}

function Update-UserPath {
    param(
        [string]$TargetDir,
        [string[]]$RemoveDirs = @()
    )

    if ($NoPathUpdate) {
        Write-Info "Skipping PATH update (-NoPathUpdate specified)."
        return
    }

    try {
        $userPath = Get-UserPath
        $pathList = $userPath -split ';' | Where-Object { $_ -ne "" }

        # Filter out directories in RemoveDirs
        if ($RemoveDirs -and $RemoveDirs.Count -gt 0) {
            $pathList = $pathList | Where-Object {
                $entry = $_.Trim().TrimEnd('\')
                $match = $false
                foreach ($rem in $RemoveDirs) {
                    if ($entry -ieq $rem.Trim().TrimEnd('\')) {
                        $match = $true
                        break
                    }
                }
                -not $match
            }
        }

        # Add TargetDir if not present
        $cleanTarget = $TargetDir.Trim().TrimEnd('\')
        $contains = $false
        foreach ($p in $pathList) {
            if ($p.Trim().TrimEnd('\') -ieq $cleanTarget) {
                $contains = $true
                break
            }
        }

        if (-not $contains) {
            $pathList = @($pathList) + $TargetDir
        }

        $newPath = $pathList -join ';'
        Set-UserPath -NewPath $newPath
        Write-Success "Updated User PATH environment variable."

        # Update current session PATH as well
        $sessionPaths = $env:Path -split ';' | Where-Object { $_ -ne "" }
        if ($sessionPaths -notcontains $TargetDir) {
            $env:Path = "$TargetDir;$env:Path"
        }
    } catch {
        Write-Warn "Could not update User PATH automatically: $_"
        Write-Warn "Please manually add '$TargetDir' to your PATH."
    }
}

function Detect-ExistingNode {
    $result = @{
        Found          = $false
        Version        = ""
        ExecutablePath = ""
        InstallDir     = ""
        NPMDir         = ""
        PathEntries    = @()
        Manager        = "Standard"
    }

    $candidates = @()

    # 1. Check where.exe / command
    $whereNode = Get-Command node.exe -ErrorAction SilentlyContinue
    if ($whereNode) {
        $candidates += $whereNode.Source
    }

    # 2. Check standard locations
    $progFiles = ${env:ProgramFiles}
    if (-not $progFiles) { $progFiles = "C:\Program Files" }
    $progFilesX86 = ${env:ProgramFiles(x86)}
    if (-not $progFilesX86) { $progFilesX86 = "C:\Program Files (x86)" }
    $appData = $env:APPDATA
    $localAppData = $env:LOCALAPPDATA

    $candidates += (Join-Path $progFiles "nodejs\node.exe")
    $candidates += (Join-Path $progFilesX86 "nodejs\node.exe")
    if ($appData) { $candidates += (Join-Path $appData "npm\node.exe") }
    if ($localAppData) { $candidates += (Join-Path $localAppData "Programs\nodejs\node.exe") }

    # 3. NVM for Windows
    if ($env:NVM_SYMLINK) { $candidates += (Join-Path $env:NVM_SYMLINK "node.exe") }
    if ($env:NVM_HOME) { $candidates += (Join-Path $env:NVM_HOME "nodejs\node.exe") }

    foreach ($cand in $candidates) {
        if (-not $cand -or -not (Test-Path $cand)) { continue }

        # Skip if already inside .uvm
        if ($cand -like "*\.uvm\*") { continue }

        try {
            $verOut = (& $cand -v 2>$null)
            if ($verOut -and $verOut.StartsWith("v")) {
                $result.Found = $true
                $result.Version = $verOut.Trim()
                $result.ExecutablePath = $cand
                $result.InstallDir = Split-Path $cand -Parent

                if ($cand -like "*nvm*" -or $env:NVM_HOME) {
                    $result.Manager = "NVM for Windows"
                } elseif ($cand -like "*Program Files*") {
                    $result.Manager = "Standard Windows Installer"
                } else {
                    $result.Manager = "Custom / System"
                }

                $paths = @($result.InstallDir)
                if ($appData) {
                    $npmPath = Join-Path $appData "npm"
                    if (Test-Path $npmPath) {
                        $result.NPMDir = $npmPath
                        $paths += $npmPath
                    }
                }
                if ($env:NVM_SYMLINK) { $paths += $env:NVM_SYMLINK }
                $result.PathEntries = $paths
                break
            }
        } catch {
            # Continue checking other candidates
        }
    }

    return $result
}

function Move-NodeToUvm {
    param(
        [hashtable]$NodeInfo,
        [string]$UvmBaseDir
    )

    $version = $NodeInfo.Version
    $sourceDir = $NodeInfo.InstallDir
    $targetDir = Join-Path $UvmBaseDir "versions\node\$version"

    Write-Info "[1/6] Copying existing Node.js ($version) to $targetDir..."
    if (-not (Test-Path $targetDir)) {
        New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    }

    Copy-Item -Path "$sourceDir\*" -Destination $targetDir -Recurse -Force

    # Verify Step 2
    Write-Info "[2/6] Verifying copied node executable..."
    $copiedNode = Join-Path $targetDir "node.exe"
    if (-not (Test-Path $copiedNode)) {
        Remove-Item -Path $targetDir -Recurse -Force -ErrorAction SilentlyContinue
        throw "Copied node executable not found at $copiedNode"
    }

    $verCheck = (& $copiedNode -v 2>$null)
    if ($verCheck -ne $version) {
        Remove-Item -Path $targetDir -Recurse -Force -ErrorAction SilentlyContinue
        throw "Copied node verification failed (expected $version, got $verCheck)"
    }

    # Backup PATH
    $prevPath = Get-UserPath

    try {
        # Update PATH
        Write-Info "[3/6] Updating User PATH..."
        $uvmBin = Join-Path $UvmBaseDir "bin"
        Update-UserPath -TargetDir $uvmBin -RemoveDirs $NodeInfo.PathEntries

        # Activate in UVM
        Write-Info "[4/6] Activating Node.js $version in UVM..."
        $uvmNodeBin = Join-Path $uvmBin "node.exe"
        Copy-Item -Path $copiedNode -Destination $uvmNodeBin -Force

        # Generate .cmd shims
        $cmdShims = @("node.cmd", "npm.cmd", "npx.cmd", "corepack.cmd")
        foreach ($shim in $cmdShims) {
            $targetShim = Join-Path $targetDir $shim
            if ($shim -eq "node.cmd") { $targetShim = $copiedNode }
            $shimPath = Join-Path $uvmBin $shim
            "@ECHO off`r`n`"$targetShim`" %*`r`n" | Out-File -FilePath $shimPath -Encoding ascii -Force
        }

        # Verify Step 5
        Write-Info "[5/6] Verifying UVM-managed Node.js..."
        $uvmVerCheck = (& $uvmNodeBin -v 2>$null)
        if ($uvmVerCheck -ne $version) {
            throw "UVM Node.js verification failed"
        }

        # Clean old files Step 6
        Write-Info "[6/6] Cleaning up previous installation..."
        try {
            Remove-Item -Path $sourceDir -Recurse -Force -ErrorAction SilentlyContinue
            if ($NodeInfo.NPMDir -and (Test-Path $NodeInfo.NPMDir)) {
                Remove-Item -Path $NodeInfo.NPMDir -Recurse -Force -ErrorAction SilentlyContinue
            }
        } catch {
            Write-Warn "Could not delete $sourceDir automatically. You may delete it manually."
        }

        Write-Success "Node.js $version was successfully migrated to UVM!"
    } catch {
        Write-Warn "Migration encountered an error: $_. Rolling back changes..."
        if ($prevPath) { Set-UserPath -NewPath $prevPath }
        Remove-Item -Path $targetDir -Recurse -Force -ErrorAction SilentlyContinue
        throw $_
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
        $userPath = Get-UserPath
        $pathList = $userPath -split ';' | Where-Object { $_ -ne "" -and $_ -ne $InstallDir }
        $cleanedPath = $pathList -join ';'
        Set-UserPath -NewPath $cleanedPath
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

    $uvmBaseDir = Split-Path $InstallDir -Parent
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
        $downloadUrls = @(
            "https://github.com/$Repo/releases/latest/download/$targetName",
            "https://github.com/onlypratyush/UVM/releases/latest/download/$targetName",
            "https://github.com/$Repo/releases/download/$Version/$targetName"
        )

        $tmpFile = [System.IO.Path]::GetTempFileName() + ".exe"
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

        foreach ($url in $downloadUrls) {
            try {
                Write-Info "Downloading $targetName from GitHub Releases ($url)..."
                Invoke-WebRequest -Uri $url -OutFile $tmpFile -UseBasicParsing -TimeoutSec 30
                if ((Test-Path $tmpFile) -and (Get-Item $tmpFile).Length -gt 10000) {
                    Move-Item -Path $tmpFile -Destination $binPath -Force
                    $installed = $true
                    break
                }
            } catch {
                # Try next URL
            }
        }

        if (Test-Path $tmpFile) { Remove-Item $tmpFile -Force -ErrorAction SilentlyContinue }

        # Fallback to Go install if present
        if (-not $installed -and (Get-Command go -ErrorAction SilentlyContinue)) {
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

    # 3. Detect existing Node.js
    $detected = Detect-ExistingNode
    if ($detected.Found) {
        Write-Host ""
        Write-Host "  ┌─────────────────────────────────────────────────────────────┐" -ForegroundColor Yellow
        Write-Host "  │ Existing Node.js Installation Found                         │" -ForegroundColor Yellow
        Write-Host "  │ Version:  $($detected.Version.PadRight(50))│" -ForegroundColor Yellow
        Write-Host "  │ Location: $($detected.InstallDir.PadRight(50))│" -ForegroundColor Yellow
        Write-Host "  │ Manager:  $($detected.Manager.PadRight(50))│" -ForegroundColor Yellow
        Write-Host "  └─────────────────────────────────────────────────────────────┘" -ForegroundColor Yellow
        Write-Host ""

        $action = $NodeAction
        if (-not $action) {
            Write-Host "How would you like UVM to handle it?"
            Write-Host "  [1] Move to UVM (Recommended) - Keep your current Node.js and let UVM manage it" -ForegroundColor Green
            Write-Host "  [2] Delete existing Node.js   - Remove existing installation"
            Write-Host "  [3] Keep existing Node.js     - Leave existing installation unchanged"
            $choice = Read-Host "Choice [1-3] (Default: 1)"

            switch ($choice) {
                "2" {
                    $confirm = Read-Host "Are you sure you want to delete existing Node.js? [y/N]"
                    if ($confirm -ieq "y" -or $confirm -ieq "yes") {
                        $action = "delete"
                        $ConfirmDelete = $true
                    } else {
                        Write-Info "Deletion cancelled. Defaulting to 'move'."
                        $action = "move"
                    }
                }
                "3" { $action = "keep" }
                Default { $action = "move" }
            }
        }

        switch ($action) {
            "move" {
                Move-NodeToUvm -NodeInfo $detected -UvmBaseDir $uvmBaseDir
            }
            "delete" {
                if ($ConfirmDelete) {
                    Write-Info "Removing existing Node.js installation at $($detected.InstallDir)..."
                    Remove-Item -Path $detected.InstallDir -Recurse -Force -ErrorAction SilentlyContinue
                    if ($detected.NPMDir) {
                        Remove-Item -Path $detected.NPMDir -Recurse -Force -ErrorAction SilentlyContinue
                    }
                    Update-UserPath -TargetDir $InstallDir -RemoveDirs $detected.PathEntries
                    Write-Success "Existing Node.js installation removed."
                } else {
                    Write-Warn "Deletion skipped (-ConfirmDelete not specified)."
                    Update-UserPath -TargetDir $InstallDir
                }
            }
            "keep" {
                Write-Info "Keeping existing Node.js installation untouched."
                Update-UserPath -TargetDir $InstallDir
            }
        }
    } else {
        Update-UserPath -TargetDir $InstallDir
    }

    Write-Host ""
    Write-Success "uvm was installed successfully to $binPath!"
    Write-Host ""
    Write-Host "Quick Start:" -ForegroundColor Cyan
    Write-Host "  1. Open a new PowerShell / Command Prompt window."
    Write-Host "  2. Verify installation: uvm --help"
    Write-Host "  3. Manage runtimes:     uvm list node"
    Write-Host "  4. Install a runtime:   uvm install node 20.11.0"
    Write-Host ""
}

if ($Uninstall) {
    Uninstall-Uvm
} else {
    Install-Uvm
}
