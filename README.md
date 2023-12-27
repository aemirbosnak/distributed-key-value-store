# CS 442 Project



# Setup

To setup server:
- Navigate to parent directory
- Run "go build ./..."

```sh
./scripts/run_system.sh

# Use `tail -F [logfilename]` to watch log output
```

No dynamic node addition implemented yet, all node connections are static

# Testing

Use `./scripts/client.sh` for most common operations.
Call it without arguments to see valid commands.

Status endpoints

```sh
curl http://localhost:3001/config
# Returns number of shards and the address of leader at every shard

curl http://localhost:8001/raft/status
# Returns raft info of node
```

# Development

- Run `go fmt ./...` to format code
- Run `go vet ./...` to lint code

