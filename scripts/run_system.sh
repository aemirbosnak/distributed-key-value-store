#!/bin/bash

LOGDIR=logs

run_configserver () {
    echo "Starting configserver nodes"
    go run ./configserver --node_id 1 --port 3001 --raft_addr localhost:13001 &> "$LOGDIR/configserver_1.log" &
    echo "configserver 1 (pid $!)"
    go run ./configserver --node_id 2 --port 3002 --raft_addr localhost:13002 &> "$LOGDIR/configserver_2.log" &
    echo "configserver 2 (pid $!)"
    go run ./configserver --node_id 3 --port 3003 --raft_addr localhost:13003 &> "$LOGDIR/configserver_3.log" &
    echo "configserver 3 (pid $!)"

    echo "Sleeping for 3 seconds as configserver nodes initialize..."
    sleep 3

    echo "Forming configserver raft cluster"
    curl -d "nodeid=2&addr=localhost:13002" "localhost:3001/raft/join" -w " (node 2 to 1)\n"
    curl -d "nodeid=3&addr=localhost:13003" "localhost:3001/raft/join" -w " (node 3 to 1)\n"

    sleep 2
    echo "configserver cluster is ready!"
}

run_dataserver () {
    SHARD_ID=$1

    echo "Starting dataserver nodes (shard #$SHARD_ID)"
    go run ./dataserver --shard_id "$SHARD_ID" --node_id 1 --port "80${SHARD_ID}1" --raft_addr "localhost:180${SHARD_ID}1" \
        &> "$LOGDIR/dataserver_${SHARD_ID}_1.log" &
    echo "dataserver 1 (shard $SHARD_ID) (pid $!)"
    go run ./dataserver --shard_id "$SHARD_ID" --node_id 2 --port "80${SHARD_ID}2" --raft_addr "localhost:180${SHARD_ID}2" \
        &> "$LOGDIR/dataserver_${SHARD_ID}_2.log" &
    echo "dataserver 2 (shard $SHARD_ID) (pid $!)"
    go run ./dataserver --shard_id "$SHARD_ID" --node_id 3 --port "80${SHARD_ID}3" --raft_addr "localhost:180${SHARD_ID}3" \
        &> "$LOGDIR/dataserver_${SHARD_ID}_3.log" &
    echo "dataserver 3 (shard $SHARD_ID) (pid $!)"

    echo "Sleeping for 3 seconds as dataserver nodes initialize..."
    sleep 3

    echo "Forming dataserver raft cluster"
    curl -d "nodeid=2&addr=localhost:180${SHARD_ID}2" "localhost:80${SHARD_ID}1/raft/join" -w " (node 2 to 1)\n"
    curl -d "nodeid=3&addr=localhost:180${SHARD_ID}3" "localhost:80${SHARD_ID}1/raft/join" -w " (node 3 to 1)\n"

    sleep 2
    echo "dataserver cluster (shard #$SHARD_ID) is ready!"
}

run_router () {
    while true; do
        echo "Starting router instance"
        go run ./router &> "$LOGDIR/router.log" &
        echo "Router instance is ready! (pid $!)"
        wait
        echo "Router instance died..."
        echo "Forcing its port to be freed (this would not happen in practice)"
        sudo ss --kill state listening src :3000
    done
}

add_shards_to_config_server () {
    echo "Adding shard leaders to configserver manually"
    curl "localhost:3001/addshard?shardID=1&shardAddress=localhost:18011" -w " (shard 1)\n"
    curl "localhost:3001/addshard?shardID=2&shardAddress=localhost:18021" -w " (shard 2)\n"
    curl "localhost:3001/addshard?shardID=3&shardAddress=localhost:18031" -w " (shard 3)\n"
}

run_all () {
    run_dataserver 1
    run_dataserver 2
    run_dataserver 3

    run_configserver
    add_shards_to_config_server
    
    (run_router) &

    wait
}

(trap "echo 'Killing all processes...'; kill 0" SIGINT; run_all)

