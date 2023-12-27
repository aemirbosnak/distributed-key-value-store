package raftserver

import (
	"github.com/hashicorp/raft"
)

type Server struct {
	raft *raft.Raft
	fsm  raft.FSM
}

func New(raft *raft.Raft, fsm raft.FSM) *Server {
	return &Server{
		raft: raft,
		fsm:  fsm,
	}
}
