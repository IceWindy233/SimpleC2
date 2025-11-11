package main

import (
	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/teamserver/data"
	"simplec2/teamserver/websocket"
)

// server is used to implement bridge.TeamServerBridgeService.
type server struct {
	bridge.UnimplementedTeamServerBridgeServiceServer
	Config *config.TeamServerConfig
	Store  data.DataStore
	Hub    *websocket.Hub
}

// NewServer creates a new server instance with the given configuration, datastore, and hub.
func NewServer(cfg *config.TeamServerConfig, store data.DataStore, hub *websocket.Hub) *server {
	return &server{Config: cfg, Store: store, Hub: hub}
}
