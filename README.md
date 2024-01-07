# Distributed Key-Value Store with Raft Consensus Algorithm

The project aims to implement a reliable and consistent in-memory key-value store in Go using the Raft consensus algorithm. It partitions data into shards using the Murmur3 hashing algorithm and employs a leader-follower design for replication among three servers per data shard.

- **Data Sharding**: Data is partitioned using Murmur3 hashing, each shard containing three replicated data servers following the leader-follower model.
- **Consistency**: Raft consensus ensures consistent data replication among servers within a shard.
- **Config Server**: A replicated config server tracks shard leader addresses and the number of active data shards. This server is also distributed with the raft algorithm to ensure the reliability of the address data.
- **Router**: The front-end interface determines the shard leader's address through the config server and directs requests accordingly.

# Setup

To setup server:
- Navigate to parent directory
- Run "go build ./..."

```sh
./scripts/run_system.sh

# Use `tail -F [logfilename]` to watch log output
```

No dynamic data shard addition implemented yet, all data shard connections are static

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

