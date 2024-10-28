# dove-rpc

This is a simple RPC proxy that implements per-IP rate-limiting, and proxies JSON-RPC calls sent to `/` to a set of providers specified in `priv/providers.json` (added to `.gitignore` to prevent key leakage).

## Running

1. Install Go: https://go.dev/doc/install
2. Clone this repository
3. Navigate to the repository directory
4. Run `go run .` to start the server
