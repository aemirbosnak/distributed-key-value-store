#!/bin/bash

NODE_ID=$1

go run ./configserver --node_id "$NODE_ID" --port "300$NODE_ID" --raft_addr "localhost:1300$NODE_ID" \
    &> "logs/configserver_$NODE_ID.log" &

echo "configserver $NODE_ID is live (pid $!)"

