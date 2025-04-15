# HTTPS Proxy Server

A lightweight HTTP proxy server that forwards requests to HTTPS servers. Specifically designed to handle forwarding HTTP requests to HTTPS endpoints reliably.

## Requirements

- Go 1.24 or later

## Quick Start

This project includes a deployment script that simplifies building and running the proxy server.

### Building

```bash
./proxy.sh build
```

This will compile the Go application and create the `https-proxy` binary.

### Running

```bash
./proxy.sh run https://api.example.com [options]
```

Options:
- `--port=NUMBER`: Port to listen on (default: 8000)
- `--verbose`: Enable detailed request/response logging
- `--insecure`: Skip TLS certificate verification (for self-signed certificates)
- `--timeout=NUMBER`: Request timeout in seconds (default: 30)

Examples:

Basic usage:
```bash
./proxy.sh run https://api.example.com
```

With custom options:
```bash
./proxy.sh run https://api.example.com --port=9000 --insecure --verbose
```

This will start the proxy server on port 9000, forwarding requests to https://api.example.com with TLS certificate verification disabled and verbose logging enabled.

### Help

```bash
./proxy.sh help
```

## Manual Usage

If you prefer to run the binary directly after building:

```bash
./https-proxy --target=https://example.com --listen=:8000 --insecure --verbose
```

## Server Configuration

For production deployment, you can set up the proxy as a system service using systemd:

1. Copy the binary to a system location:
   ```bash
   sudo cp ./https-proxy /usr/local/bin/
   ```

2. Create a systemd service file:
   ```bash
   sudo nano /etc/systemd/system/https-proxy.service
   ```
   
   With content like:
   ```
   [Unit]
   Description=HTTPS Proxy Server
   After=network.target

   [Service]
   Type=simple
   User=nobody
   Group=nogroup
   ExecStart=/usr/local/bin/https-proxy --target=https://your-target-server.com --listen=:8000
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

3. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable https-proxy
   sudo systemctl start https-proxy
   ```

4. Check status and logs:
   ```bash
   sudo systemctl status https-proxy
   sudo journalctl -u https-proxy
   ```

## Troubleshooting

### Common Issues with HTTPS Connections

- **Certificate Errors**: If the target HTTPS server uses a self-signed certificate or a certificate not trusted by your system, use the `--insecure` flag.
- **Connection Failures**: Ensure the target server is accessible from your network and that any firewalls permit outbound connections.
- **HTTP vs HTTPS**: This proxy is specifically designed to forward to HTTPS endpoints. Make sure your target URL starts with `https://`.