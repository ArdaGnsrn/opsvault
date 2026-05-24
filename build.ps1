param(
    [string]$Target = "linux"
)

$VersionRaw = git describe --tags --always --dirty 2>$null
$Version = if ($VersionRaw) { $VersionRaw } else { "dev" }

$CommitRaw = git rev-parse --short HEAD 2>$null
$Commit = if ($CommitRaw) { $CommitRaw } else { "unknown" }

$Ldflags = "-X github.com/ArdaGnsrn/opsvault/internal/buildinfo.Version=$Version " +
           "-X github.com/ArdaGnsrn/opsvault/internal/buildinfo.Commit=$Commit " +
           "-w -s"

New-Item -ItemType Directory -Force -Path dist | Out-Null

switch ($Target) {
    "windows" {
        Write-Host "Building for Windows..."
        go build -ldflags $Ldflags -o dist\opsvault.exe .
        Write-Host "-> dist\opsvault.exe"
    }
    "linux" {
        Write-Host "Building for Linux (amd64)..."
        $env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
        go build -ldflags $Ldflags -o dist\opsvault-linux-amd64 .
        $env:GOOS = ""; $env:GOARCH = ""; $env:CGO_ENABLED = ""
        Write-Host "-> dist\opsvault-linux-amd64"
    }
    "linux-arm64" {
        Write-Host "Building for Linux (arm64)..."
        $env:GOOS = "linux"; $env:GOARCH = "arm64"; $env:CGO_ENABLED = "0"
        go build -ldflags $Ldflags -o dist\opsvault-linux-arm64 .
        $env:GOOS = ""; $env:GOARCH = ""; $env:CGO_ENABLED = ""
        Write-Host "-> dist\opsvault-linux-arm64"
    }
    default {
        Write-Error "Unknown target: $Target. Use: windows, linux, linux-arm64"
        exit 1
    }
}
