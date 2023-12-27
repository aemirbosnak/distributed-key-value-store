#!/bin/bash

SHARD_ID=$1
NODE_ID=$2

go run ./dataserver --shard_id "$SHARD_ID" --node_id "$NODE_ID" --port "80$SHARD_ID$NODE_ID" \
    --raft_addr "localhost:180$SHARD_ID$NODE_ID" &> "logs/dataserver_${SHARD_ID}_$NODE_ID.log" &

echo "dataserver $NODE_ID is live (shard $SHARD_ID) (pid $!)"

