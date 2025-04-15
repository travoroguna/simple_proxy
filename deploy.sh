#!/bin/bash

# Build script for proxy server

BUILD_DIR="./build"
BINARY_NAME="proxy-server"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
  echo -e "${GREEN}[+] $1${NC}"
}

print_error() {
  echo -e "${RED}[!] $1${NC}"
}

# Create build directory if it doesn't exist
if [ ! -d "$BUILD_DIR" ]; then
  print_status "Creating build directory"
  mkdir -p "$BUILD_DIR"
fi

# Build function
build() {
  print_status "Building proxy server"
  go build -o "$BUILD_DIR/$BINARY_NAME" .
  if [ $? -eq 0 ]; then
    print_status "Build successful: $BUILD_DIR/$BINARY_NAME"
  else
    print_error "Build failed"
    exit 1
  fi
}

# Start function
start() {
  local TARGET=${1:-"http://localhost:8080"}
  local PORT=${2:-"8000"}
  
  print_status "Starting proxy server on port $PORT pointing to $TARGET"
  "$BUILD_DIR/$BINARY_NAME" --target="$TARGET" --listen=":$PORT"
}

# Display help
show_help() {
  echo "Usage: $0 [command] [options]"
  echo ""
  echo "Commands:"
  echo "  build               Build the proxy server"
  echo "  start [url] [port]  Start the proxy server"
  echo "                      - url: Target server URL (default: http://localhost:8080)"
  echo "                      - port: Port to listen on (default: 8000)"
  echo "  help                Show this help message"
}

# Main script logic
case "$1" in
  "build")
    build
    ;;
  "start")
    if [ ! -f "$BUILD_DIR/$BINARY_NAME" ]; then
      print_status "Binary not found, building first"
      build
    fi
    start "$2" "$3"
    ;;
  "help" | *)
    show_help
    ;;
esac