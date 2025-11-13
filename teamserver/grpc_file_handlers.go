package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"simplec2/pkg/bridge"
)

func (s *server) GetTaskedFileChunk(ctx context.Context, in *bridge.GetTaskedFileChunkRequest) (*bridge.GetTaskedFileChunkResponse, error) {
	// For now, we don't have beacon identity in the gRPC context.
	// This is a security risk that needs to be addressed later by passing the beacon ID
	// from the listener's mTLS certificate subject.
	// TODO: Add beacon identity to gRPC context and verify task ownership.

	task, err := s.Store.GetTask(in.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	if task.Command != "download" {
		return nil, status.Errorf(codes.PermissionDenied, "task is not a download task")
	}

	var downloadArgs struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(task.Arguments), &downloadArgs); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse download arguments for task %s: %v", task.TaskID, err)
	}

	sourcePath := downloadArgs.Source
	// Security Check: Ensure the final path is within the intended uploads directory.
	absUploadsDir, err := filepath.Abs(s.Config.UploadsDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not resolve uploads directory")
	}
	absFilePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not resolve file path")
	}
	if !strings.HasPrefix(absFilePath, absUploadsDir) {
		return nil, status.Errorf(codes.PermissionDenied, "access denied: file is outside of the uploads directory")
	}

	file, err := os.Open(absFilePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to open file: %v", err)
	}
	defer file.Close()

	chunkBuffer := make([]byte, ChunkSize)
	offset := int64(in.ChunkNumber) * ChunkSize

	bytesRead, err := file.ReadAt(chunkBuffer, offset)
	if err != nil && err != io.EOF {
		return nil, status.Errorf(codes.Internal, "failed to read chunk: %v", err)
	}

	return &bridge.GetTaskedFileChunkResponse{
		ChunkData: chunkBuffer[:bytesRead],
	}, nil
}
