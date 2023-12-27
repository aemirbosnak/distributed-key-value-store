package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"

	"github.com/vedxyz/cs442-project/raftserver"
	"github.com/vedxyz/cs442-project/raftserver/fsm"
)

type ConfigServer struct {
	raft   *raft.Raft
	server *raftserver.Server
	fsm    raft.FSM
}

const (
	tcpTimeout    = 1 * time.Second
	snapInterval  = 30 * time.Second
	snapThreshold = 1000
)

var (
	nodeID   = flag.String("node_id", "node_1", "raft node id")
	port     = flag.Int("port", 3001, "http port")
	raftaddr = flag.String("raft_addr", "localhost:13001", "raft address")
	storedir = flag.String("store_dir", "", "db dir")
)

func NewConfigServer(raft *raft.Raft, fsm raft.FSM) *ConfigServer {
	server := raftserver.New(raft, fsm)
	return &ConfigServer{
		raft:   raft,
		server: server,
		fsm:    fsm,
	}
}

func (cs *ConfigServer) ConfigHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("[HTTP] config is requested")

	fsmInstance, ok := cs.fsm.(*fsm.FSM)
	if !ok {
		http.Error(w, "error casting to FSM", http.StatusInternalServerError)
		return
	}

	response := make(map[string]interface{})

	key := 1
	for {
		keyStr := strconv.Itoa(key)
		value, err := fsmInstance.Get(keyStr)
		if err != nil {
			break
		}

		response[keyStr] = value
		key++
	}

	response["shardCount"] = strconv.Itoa(key - 1)

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "error creating JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func (cs *ConfigServer) AddShardHandler(w http.ResponseWriter, r *http.Request) {
	shardID := r.FormValue("shardID")
	shardAddress := r.FormValue("shardAddress")

	payload := fsm.Payload{
		OP:    fsm.PUT,
		Key:   shardID,
		Value: shardAddress,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applyFuture := cs.raft.Apply(data, 500*time.Millisecond)
	if err := applyFuture.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := applyFuture.Response().(*fsm.ApplyResponse)
	if !ok {
		w.Write([]byte("error raft response"))
		return
	}

	w.Write([]byte("ok"))
}

func (cs *ConfigServer) NewLeaderHandler(w http.ResponseWriter, r *http.Request) {
	shardID := r.FormValue("shardID")
	shardAddress := r.FormValue("shardAddress")

	// Process the new leader info
	fmt.Printf("New leader address: %s, shard ID: %s\n", shardAddress, shardID)

	payload := fsm.Payload{
		OP:    fsm.PUT,
		Key:   shardID,
		Value: shardAddress,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applyFuture := cs.raft.Apply(data, 500*time.Millisecond)
	if err := applyFuture.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := applyFuture.Response().(*fsm.ApplyResponse)
	if !ok {
		w.Write([]byte("error raft response"))
		return
	}

	w.Write([]byte("ok"))
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

	configServer := NewConfigServer(raftServer, fsmStore)

	http.HandleFunc("/config", configServer.ConfigHandler)
	http.HandleFunc("/addshard", configServer.AddShardHandler)
	http.HandleFunc("/newleader", configServer.NewLeaderHandler)

	http.HandleFunc("/raft/join", configServer.server.RaftJoin)
	http.HandleFunc("/raft/status", configServer.server.RaftStatus)
	http.HandleFunc("/raft/leave", configServer.server.RaftLeave)

	fmt.Printf("Config Server listening on port %d\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
