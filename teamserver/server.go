package main

import (
	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/teamserver/data"
	"simplec2/teamserver/service"
	"simplec2/teamserver/websocket"
)

// server is used to implement bridge.TeamServerBridgeService.
type server struct {
	bridge.UnimplementedTeamServerBridgeServiceServer
	Config          *config.TeamServerConfig
	Store           data.DataStore
	Hub             *websocket.Hub
	ListenerService service.ListenerService
	PortFwdService  service.PortFwdService // Add PortFwdService
}

// NewServer creates a new server instance with the given configuration, datastore, hub, and services.
func NewServer(cfg *config.TeamServerConfig, store data.DataStore, hub *websocket.Hub, listenerService service.ListenerService, portFwdService service.PortFwdService) *server {
	return &server{Config: cfg, Store: store, Hub: hub, ListenerService: listenerService, PortFwdService: portFwdService}
}
