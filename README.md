# Demo HTTP Server (Go) — fixed

Running locally:
1. Copy the configuration example:
   cp config.example.yaml config.yaml

2. Set the secrets via environment variables (recommended):
   export DB_CONNECTION="postgres://user:password@host:port/db"
   export EXTERNAL_API_KEY="your_key_here"

3. Install the dependencies and run:
   go mod tidy
   go run main.go

Important: DO NOT commit files with secrets (config.yaml, .env) to the repository.