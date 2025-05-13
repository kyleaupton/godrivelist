#!/bin/bash

# Simple build script for godrivelist
# Detects platform and builds accordingly

# Get the current OS
OS=$(uname -s)

# Set CGO enabled for all platforms
export CGO_ENABLED=1

case "$OS" in
    "Darwin")  # macOS
        echo "Building for macOS..."
        # Check if Xcode CLI tools are installed
        if ! command -v xcode-select &> /dev/null; then
            echo "Error: Xcode Command Line Tools not found"
            echo "Please install them with: xcode-select --install"
            exit 1
        fi
        go build -v ./...
        ;;
    "Linux")  # Linux
        echo "Building for Linux..."
        go build -v ./...
        ;;
    "MINGW"*|"MSYS"*|"CYGWIN"*)  # Windows through MSYS2, MinGW, or Cygwin
        echo "Building for Windows (MSYS/MinGW)..."
        go build -v ./...
        ;;
    *)
        echo "Unsupported operating system: $OS"
        echo "Please build manually with: go build"
        exit 1
        ;;
esac

# Build the example
echo "Building example..."
go build -o example/godrivelist-example example/main.go

echo "Build complete!" 