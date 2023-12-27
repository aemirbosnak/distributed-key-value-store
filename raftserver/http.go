package raftserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/vedxyz/cs442-project/raftserver/fsm"
)

func (s *Server) PutHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	val := r.Form.Get("val")
	if key == "" || val == "" {
		http.Error(w, "error key or val is empty", http.StatusOK)
		return
	}
    log.Printf("[HTTP-PUT] key %s was put into this node", key)

	payload := fsm.Payload{
		OP:    fsm.PUT,
		Key:   key,
		Value: val,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applyFuture := s.raft.Apply(data, 500*time.Millisecond)
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

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	if key == "" {
		http.Error(w, "error key is empty", http.StatusOK)
		return
	}

	fsmInstance, ok := s.fsm.(*fsm.FSM)
	if !ok {
		http.Error(w, "error casting to FSM", http.StatusInternalServerError)
		return
	}

	value, err := fsmInstance.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
    log.Printf("[HTTP-GET] key %s was found on this node", key)

	var valueBytes []byte
	if valueStr, ok := value.(string); ok {
		valueBytes = []byte(valueStr)
	} else if valueBytes, ok = value.([]byte); !ok {
		http.Error(w, "error: value conversion failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write(valueBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	if key == "" {
		http.Error(w, "error key is empty", http.StatusOK)
		return
	}
    log.Printf("[HTTP-DELETE] key %s was found on this node", key)

	payload := fsm.Payload{
		OP:  fsm.DEL,
		Key: key,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applyFuture := s.raft.Apply(data, 500*time.Millisecond)
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
