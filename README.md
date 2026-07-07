# 🚀 WeMon Agent

The Go-based, lightweight monitoring agent for the WeMon Server Monitoring Stack. It streams real-time system metrics (CPU, Memory, Disk, Network) to the WeMon backend via a persistent secure WebSocket connection.

---

## 🛠️ Key Features

- **Lightweight**: Consumes minimal CPU and RAM.
- **WebSocket Streaming**: Single persistent connection for sub-second, bidirectional metric updates (avoids HTTP polling overhead).
- **Auto-Reconnect**: Robust reconnection logic with exponential backoff if the network drops or the server restarts.
- **Dynamic Configuration**: Responds to real-time configuration changes (like monitoring interval) pushed directly from the server.
- **Zero Dependencies**: Compiles down to a single self-contained binary file.

---

## 📦 Production Installation

Simply run the 1-line installer on the target Linux machine:

```bash
curl -sL https://raw.githubusercontent.com/wylayss/WeMon-Agent/main/installer/install.sh | sudo bash -s -- --server https://wemon.werix.net --token YOUR_NODE_TOKEN
```

### Checking Agent Status
```bash
# Check if the service is running
systemctl status wemon-agent.service

# View live system logs for the agent
journalctl -u wemon-agent.service -f
```

---

## ⚙️ Development & Testing

### Prerequisites
- [Go 1.22+](https://go.dev/dl/) installed.

### Local Execution
1. Install dependencies:
   ```bash
   go mod tidy
   ```
2. Create a local `config.json` file in the root directory:
   ```json
   {
     "server_url": "http://localhost:3001",
     "node_token": "wmn_tok_your_node_token",
     "interval_seconds": 2,
     "insecure_skip_verify": true
   }
   ```
3. Run the agent locally:
   ```bash
   go run main.go
   ```

### Building the Binaries
To build the optimized production binary for the current system architecture:
```bash
go build -ldflags="-s -w" -o wemon-agent
```

To cross-compile binaries for other architectures:
```bash
# Compile for Linux 64-bit AMD
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o wemon-agent-linux-amd64

# Compile for Linux 64-bit ARM
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o wemon-agent-linux-arm64
```
