package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"

	server "github.com/vedxyz/cs442-project/raftserver"
	"github.com/vedxyz/cs442-project/raftserver/fsm"
)

const (
	tcpTimeout    = 1 * time.Second
	snapInterval  = 30 * time.Second
	snapThreshold = 1000
)

var (
	nodeID   = flag.String("node_id", "node_1", "raft node id")
	port     = flag.Int("port", 8001, "http port")
	raftaddr = flag.String("raft_addr", "localhost:18001", "raft address")
	shardID  = flag.Int("shard_id", 1, "shard id")
	storedir = flag.String("store_dir", "", "db dir")
)

func LeaderObserver(raftInstance *raft.Raft, configServerURL string) {
	go func() {
		lastAddress := raftInstance.Leader()
		for {
			currentAddress := raftInstance.Leader()
			if currentAddress != lastAddress {
				lastAddress = currentAddress

				// Check if this node is the leader
				if raftInstance.State() == raft.Leader {
					// Send request to the config server only if this node is the leader
					newLeaderInfo := fmt.Sprintf("shardID=%d&shardAddress=%s", *shardID, currentAddress)
					resp, err := http.Post(configServerURL+"/newleader", "application/x-www-form-urlencoded", strings.NewReader(newLeaderInfo))
					if err != nil {
						fmt.Println("Error sending request to config server:", err)
					} else {
						defer resp.Body.Close()
						body, _ := io.ReadAll(resp.Body)
						fmt.Println("Config server response:", string(body))
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	dir := *storedir
	if dir != "" {
		log.Println("Using existing store_dir: ", dir)
	} else {
		log.Println("Creating temp dir for raft")
		tempDir, err := os.MkdirTemp("", "cs442_")
		if err != nil {
			log.Fatalln("Failed to create temp dir")
		}
		defer os.RemoveAll(tempDir)
		log.Printf("Created temp dir %s", tempDir)
		dir = tempDir
	}

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(*nodeID)
	raftConfig.SnapshotInterval = snapInterval
	raftConfig.SnapshotThreshold = snapThreshold

	fsmStore := fsm.NewFSM()

	// Raft configuration
	store, err := raftboltdb.NewBoltStore(filepath.Join(dir, "raft.db"))
	if err != nil {
		log.Fatal(err)
	}

	cacheStore, err := raft.NewLogCache(256, store)
	if err != nil {
		log.Fatal(err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dir, 1, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", *raftaddr)
	if err != nil {
		log.Fatal(err)
	}

	transport, err := raft.NewTCPTransport(*raftaddr, tcpAddr, 3, tcpTimeout, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	raftServer, err := raft.NewRaft(raftConfig, fsmStore, cacheStore, store, snapshotStore, transport)
	if err != nil {
		log.Fatal(err)
	}

	raftServer.BootstrapCluster(raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(*nodeID),
				Address: transport.LocalAddr(),
			},
		},
	})

	server := server.New(raftServer, fsmStore)
	go LeaderObserver(raftServer, "http://127.0.0.1:3001")

	http.HandleFunc("/get", server.GetHandler)
	http.HandleFunc("/put", server.PutHandler)
	http.HandleFunc("/delete", server.DeleteHandler)

	http.HandleFunc("/raft/join", server.RaftJoin)
	http.HandleFunc("/raft/status", server.RaftStatus)
	http.HandleFunc("/raft/leave", server.RaftLeave)

	fmt.Println("Server listening on: ", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
