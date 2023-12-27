package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/twmb/murmur3"
)

const HASH_MODULO = 16384

var (
	port              = flag.Int("port", 3000, "http port")
	configserverport1 = flag.Int("configserverport1", 3001, "config server port 1")
	configserverport2 = flag.Int("configserverport2", 3002, "config server port 2")
	configserverport3 = flag.Int("configserverport3", 3003, "config server port 3")
)

func hashStringKey(key string) uint64 {
	return murmur3.StringSum64(key)
}

func getShardIndexFromHash(hash uint64, shardCount int) int {
	reducedHash := hash % HASH_MODULO
	bucketSize := HASH_MODULO / shardCount //TODO: add div by zero check

	for i := 0; i < shardCount; i++ {
		if reducedHash < uint64((i+1)*bucketSize) {
			return i
		}
	}
	log.Fatalln("Failed to place key into a shard's bucket")
	return -1
}

func getShardIndexFromStringKey(key string, shardCount int) int {
	return getShardIndexFromHash(hashStringKey(key), shardCount)
}

func getShardCount() (int, error) {
    ports := [3]int { *configserverport1, *configserverport2, *configserverport3 }
    var resp *http.Response
    var err error
    for _, *port = range ports {
        resp, err = http.Get(fmt.Sprintf("http://localhost:%d/config", *port))
        if err == nil {
            log.Printf("Using configserver at port %d", *port)
            defer resp.Body.Close()
            break
        }
    }
    if err != nil {
        log.Println("Failed to fallback to any configserver")
        return 0, err
    }

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	shardCountStr, ok := response["shardCount"].(string)
	if !ok {
		return 0, err
	}

	shardCount, err := strconv.Atoi(shardCountStr)
	if err != nil {
		return 0, fmt.Errorf("error converting shardCount to int: %v", err)
	}

	return shardCount, nil
}

func getShardAddress(shardID int) (string, error) {
    ports := [3]int { *configserverport1, *configserverport2, *configserverport3 }
    var resp *http.Response
    var err error
    for _, *port = range ports {
        resp, err = http.Get(fmt.Sprintf("http://localhost:%d/config", *port))
        if err == nil {
            defer resp.Body.Close()
            break
        }
    }
    if err != nil {
        log.Println("Failed to fallback to any configserver")
        return "", err
    }

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	leaderAddressRaw, exists := response[strconv.Itoa(shardID)].(string)
	if !exists {
		return "", err
	}

	// Extracting the port number from the leader address
	parts := strings.Split(leaderAddressRaw, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid leader address format: %s", leaderAddressRaw)
	}

	// Parsing the port
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("error parsing port: %v", err)
	}

	// Converting the port number from Raft to server format
	serverPort := port - 10000

	// Building the modified leader address
	modifiedAddress := strings.Replace(leaderAddressRaw, parts[1], strconv.Itoa(serverPort), 1)

	return modifiedAddress, nil
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	shardCount, err := getShardCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard count: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprint(shardCount)))
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")

	if key == "" {
		http.Error(w, "error key is empty", http.StatusBadRequest)
		return
	}

	shardCount, err := getShardCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard count: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	shardIndex := getShardIndexFromStringKey(key, shardCount)

	shardAddress, err := getShardAddress(shardIndex + 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard address: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	serverURL := fmt.Sprintf("http://%s/get?key=%s", shardAddress, key)

	resp, err := http.Get(serverURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("error contacting server node: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("val")

	if key == "" || value == "" {
		http.Error(w, "error key or value is empty", http.StatusBadRequest)
		return
	}

	shardCount, err := getShardCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard count: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	shardIndex := getShardIndexFromStringKey(key, shardCount)

	shardAddress, err := getShardAddress(shardIndex + 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard address: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	serverURL := fmt.Sprintf("http://%s/put?key=%s&val=%s", shardAddress, key, value)

	resp, err := http.Post(serverURL, "text/plain", nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("error contacting server node: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")

	if key == "" {
		http.Error(w, "error key is empty", http.StatusBadRequest)
		return
	}

	shardCount, err := getShardCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard count: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	shardIndex := getShardIndexFromStringKey(key, shardCount)

	shardAddress, err := getShardAddress(shardIndex + 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting shard address: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	serverURL := fmt.Sprintf("http://%s/delete?key=%s", shardAddress, key)

	req, err := http.NewRequest("DELETE", serverURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating DELETE request: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("error contacting server node: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	log.Println("Starting router node")

	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/put", putHandler)
	http.HandleFunc("/delete", deleteHandler)

	fmt.Println("Server listening on: ", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
