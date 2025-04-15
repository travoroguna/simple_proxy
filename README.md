# Simple HTTP Proxy Server

A lightweight HTTP proxy server that forwards requests to a target server.

## Requirements

- Go 1.24 or later

## Deployment

This project includes a deployment script that simplifies building and running the proxy server.

### Building

```bash
./deploy.sh build
```

This will compile the Go application and place the binary in the `./build` directory.

### Running

```bash
./deploy.sh start [target_url] [port]
```

Parameters:
- `target_url`: The URL of the server to proxy requests to (default: http://localhost:8080)
- `port`: The port to listen on (default: 8000)

Example:
```bash
./deploy.sh start https://api.example.com 9000
```

This will start the proxy server on port 9000, forwarding requests to https://api.example.com.

### Help

```bash
./deploy.sh help
```

## Manual Usage

If you prefer to run the binary directly:

```bash
./build/proxy-server --target=https://example.com --listen=:8000
```

## Server Configuration

For production deployment, you can set up the proxy as a system service using systemd:

1. Copy the binary to a system location:
   ```bash
   sudo cp ./build/proxy-server /usr/local/bin/
   ```

2. Edit the systemd service file to set your target server and port:
   ```bash
   sudo cp proxy-server.service /etc/systemd/system/
   sudo nano /etc/systemd/system/proxy-server.service
   ```
   
   Update the `ExecStart` line with the correct path and parameters.

3. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable proxy-server
   sudo systemctl start proxy-server
   ```

4. Check status:
   ```bash
   sudo systemctl status proxy-server
   ```

5. View logs:
   ```bash
   sudo journalctl -u proxy-server
   ```