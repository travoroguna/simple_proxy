# HTTPS Proxy Server

A lightweight HTTP proxy server that forwards requests to HTTPS servers. Specifically designed to handle forwarding HTTP requests to HTTPS endpoints reliably. Now with multi-target support!

## Requirements

- Go 1.24 or later

## Quick Start

This project includes a deployment script that simplifies building and running the proxy server.

### Building

```bash
./proxy.sh build
```

This will compile the Go application and create the `https-proxy` binary.

### Running in Single-Target Mode

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

### Running in Multi-Target Mode

Multi-target mode allows you to route different path prefixes to different backend servers.

1. Create a config file:
```bash
./proxy.sh create-config
```

2. Edit the generated `config.json` file with your actual targets:
```json
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
```

3. Run with the config file:
```bash
./proxy.sh run-config config.json [options]
```

Options:
- `--port=NUMBER`: Port to listen on (overrides config file)
- `--verbose`: Enable detailed request/response logging (overrides config file)
- `--timeout=NUMBER`: Request timeout in seconds (overrides config file)

### Help

```bash
./proxy.sh help
```

## Configuration Options

### Target Configuration

Each target in multi-target mode has the following options:

- `path`: URL path prefix to match (e.g., "/api/v1")
- `targetUrl`: Target HTTPS server URL
- `insecure`: Whether to skip TLS certificate verification
- `stripPrefix`: Whether to strip the path prefix before forwarding

Path matching is based on the most specific match. For example, with targets for `/api/v1` and `/`, a request to `/api/v1/users` will be routed to the `/api/v1` target.

When `stripPrefix` is set to `true`, the path prefix is removed before forwarding. For example, a request to `/api/v1/users` will be forwarded as `/users` to the target server.

## Manual Usage

If you prefer to run the binary directly after building:

Single-target mode:
```bash
./https-proxy --target=https://example.com --listen=:8000 --insecure --verbose
```

Multi-target mode:
```bash
./https-proxy --config=config.json --listen=:8000 --verbose
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
   
   With content like (for multi-target mode):
   ```
   [Unit]
   Description=HTTPS Proxy Server
   After=network.target

   [Service]
   Type=simple
   User=nobody
   Group=nogroup
   ExecStart=/usr/local/bin/https-proxy --config=/etc/https-proxy/config.json
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

- **Certificate Errors**: If the target HTTPS server uses a self-signed certificate or a certificate not trusted by your system, use the `insecure` option.
- **Connection Failures**: Ensure the target server is accessible from your network and that any firewalls permit outbound connections.
- **HTTP vs HTTPS**: This proxy is specifically designed to forward to HTTPS endpoints. Make sure your target URL starts with `https://`.
- **Path Mapping Issues**: In multi-target mode, ensure your path prefixes are correctly defined and do not conflict with each other. More specific paths should be defined before less specific ones.