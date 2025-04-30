#!/bin/bash

# HTTPS proxy deployment script with multi-target support

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

# Run the proxy in legacy single-target mode
run_legacy() {
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

# Run the proxy with a config file (multi-target mode)
run_with_config() {
  # Check if built
  if [ ! -f "./https-proxy" ]; then
    print_warn "Proxy not built yet, building now..."
    build
  fi
  
  # Validate required arguments
  if [ -z "$1" ]; then
    print_error "Missing config file path. Usage: $0 run-config config.json [options]"
  fi
  
  CONFIG_FILE=$1
  shift # Remove the first argument (config file)
  
  # Check if config file exists
  if [ ! -f "$CONFIG_FILE" ]; then
    print_error "Config file not found: $CONFIG_FILE"
  fi
  
  # Default settings
  PORT=8000
  VERBOSE=false
  TIMEOUT=30
  
  # Parse remaining options
  while [[ "$#" -gt 0 ]]; do
    case $1 in
      --port=*) PORT="${1#*=}" ;;
      --verbose) VERBOSE=true ;;
      --timeout=*) TIMEOUT="${1#*=}" ;;
      *) print_warn "Unknown parameter: $1" ;;
    esac
    shift
  done
  
  # Build command
  CMD="./https-proxy --config=$CONFIG_FILE --listen=:$PORT --timeout=$TIMEOUT"
  
  if [ "$VERBOSE" = true ]; then
    CMD="$CMD --verbose"
  fi
  
  # Start the proxy
  print_info "Starting multi-target proxy on port $PORT using config: $CONFIG_FILE"
  print_info "Command: $CMD"
  $CMD
}

# Create a sample config file
create_config() {
  CONFIG_FILE="config.json"
  
  if [ -f "$CONFIG_FILE" ]; then
    read -p "File $CONFIG_FILE already exists. Overwrite? (y/n): " CONFIRM
    if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
      print_info "Operation cancelled"
      return
    fi
  fi
  
  cat > "$CONFIG_FILE" << EOF
{
  "listen": ":8000",
  "timeout": 30,
  "verbose": false,
  "targets": [
    {
      "path": "/api/v1",
      "targetUrl": "https://api-v1.example.com",
      "insecure": false,
      "stripPrefix": true
    },
    {
      "path": "/api/v2",
      "targetUrl": "https://api-v2.example.com",
      "insecure": false,
      "stripPrefix": true
    },
    {
      "path": "/",
      "targetUrl": "https://default.example.com",
      "insecure": false,
      "stripPrefix": false
    }
  ]
}
EOF
  
  print_info "Created sample config file: $CONFIG_FILE"
  print_info "Edit this file with your actual target URLs and settings"
}

# Show help
show_help() {
  echo "HTTPS Proxy Server"
  echo
  echo "Usage:"
  echo "  $0 build                      Build the proxy server"
  echo "  $0 run TARGET_URL [options]   Run in single-target mode"
  echo "  $0 run-config CONFIG [options] Run in multi-target mode with config file"
  echo "  $0 create-config              Create a sample config file"
  echo
  echo "Single-target Options:"
  echo "  --port=NUMBER       Port to listen on (default: 8000)"
  echo "  --verbose           Enable verbose logging of requests and responses"
  echo "  --insecure          Skip TLS certificate verification (for self-signed certs)"
  echo "  --timeout=NUMBER    Request timeout in seconds (default: 30)"
  echo
  echo "Multi-target Options:"
  echo "  --port=NUMBER       Port to listen on (overrides config file)"
  echo "  --verbose           Enable verbose logging (overrides config file)"
  echo "  --timeout=NUMBER    Request timeout in seconds (overrides config file)"
  echo
  echo "Examples:"
  echo "  $0 run https://api.example.com"
  echo "  $0 run https://api.example.com --port=9000 --insecure --verbose"
  echo "  $0 run-config config.json --verbose"
  echo
  echo "Config file format (JSON):"
  echo '{
  "listen": ":8000",
  "timeout": 30,
  "verbose": false,
  "targets": [
    {
      "path": "/api/v1",
      "targetUrl": "https://api-v1.example.com",
      "insecure": false,
      "stripPrefix": true
    },
    {
      "path": "/",
      "targetUrl": "https://default.example.com",
      "insecure": false,
      "stripPrefix": false
    }
  ]
}'
}

# Main command processing
case "$1" in
  build)
    build
    ;;
  run)
    shift # Remove 'run' argument
    run_legacy "$@"
    ;;
  run-config)
    shift # Remove 'run-config' argument
    run_with_config "$@"
    ;;
  create-config)
    create_config
    ;;
  help|--help|-h)
    show_help
    ;;
  *)
    print_error "Unknown command. Use '$0 help' for usage information."
    ;;
esac