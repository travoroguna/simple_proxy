#!/bin/bash

# Simple HTTPS proxy deployment script

# Colors for better output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Print functions
print_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
  echo -e "${RED}[ERROR]${NC} $1"
  exit 1
}

# Build the proxy
build() {
  print_info "Building HTTPS proxy server..."
  go build -o https-proxy
  if [ $? -ne 0 ]; then
    print_error "Build failed"
  fi
  print_info "Build successful"
}

# Run the proxy
run() {
  # Check if built
  if [ ! -f "./https-proxy" ]; then
    print_warn "Proxy not built yet, building now..."
    build
  fi
  
  # Validate required arguments
  if [ -z "$1" ]; then
    print_error "Missing target URL. Usage: $0 run https://target-server.com [options]"
  fi
  
  TARGET_URL=$1
  shift # Remove the first argument (target URL)
  
  # Default settings
  PORT=8000
  VERBOSE=false
  INSECURE=false
  TIMEOUT=30
  
  # Parse remaining options
  while [[ "$#" -gt 0 ]]; do
    case $1 in
      --port=*) PORT="${1#*=}" ;;
      --verbose) VERBOSE=true ;;
      --insecure) INSECURE=true ;;
      --timeout=*) TIMEOUT="${1#*=}" ;;
      *) print_warn "Unknown parameter: $1" ;;
    esac
    shift
  done
  
  # Build command
  CMD="./https-proxy --target=$TARGET_URL --listen=:$PORT --timeout=$TIMEOUT"
  
  if [ "$VERBOSE" = true ]; then
    CMD="$CMD --verbose"
  fi
  
  if [ "$INSECURE" = true ]; then
    CMD="$CMD --insecure"
    print_warn "TLS certificate verification is disabled (insecure mode)"
  fi
  
  # Start the proxy
  print_info "Starting proxy on port $PORT forwarding to $TARGET_URL"
  print_info "Command: $CMD"
  $CMD
}

# Show help
show_help() {
  echo "HTTPS Proxy Server"
  echo
  echo "Usage:"
  echo "  $0 build                    Build the proxy server"
  echo "  $0 run TARGET_URL [options] Run the proxy server"
  echo
  echo "Options:"
  echo "  --port=NUMBER       Port to listen on (default: 8000)"
  echo "  --verbose           Enable verbose logging of requests and responses"
  echo "  --insecure          Skip TLS certificate verification (for self-signed certs)"
  echo "  --timeout=NUMBER    Request timeout in seconds (default: 30)"
  echo
  echo "Examples:"
  echo "  $0 run https://api.example.com"
  echo "  $0 run https://api.example.com --port=9000 --insecure --verbose"
}

# Main command processing
case "$1" in
  build)
    build
    ;;
  run)
    shift # Remove 'run' argument
    run "$@"
    ;;
  help|--help|-h)
    show_help
    ;;
  *)
    print_error "Unknown command. Use '$0 help' for usage information."
    ;;
esac