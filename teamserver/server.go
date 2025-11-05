package main

import (
	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/teamserver/data"
)

// server is used to implement bridge.TeamServerBridgeService.
type server struct {
	bridge.UnimplementedTeamServerBridgeServiceServer
	Config *config.TeamServerConfig
	Store  data.DataStore
}

// NewServer creates a new server instance with the given configuration and datastore.
func NewServer(cfg *config.TeamServerConfig, store data.DataStore) *server {
	return &server{Config: cfg, Store: store}
}
