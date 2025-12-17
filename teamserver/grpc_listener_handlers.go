package main

import (
	"context"
	"fmt"
	"io"

	"simplec2/pkg/bridge"
	"simplec2/pkg/logger"
)

func (s *server) ListenerControl(stream bridge.TeamServerBridgeService_ListenerControlServer) error {
	// 1. 读取第一条消息以获取 Listener 名称
	statusMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial status: %w", err)
	}

	listenerName := statusMsg.ListenerName
	if listenerName == "" {
		return fmt.Errorf("listener name is empty in initial status")
	}

	// Auto-Register: Ensure listener exists in DB
	ctx := context.Background()
	if _, err := s.ListenerService.GetListener(ctx, listenerName); err != nil {
		// If not found (or DB error), try to create if we have type info
		if statusMsg.Type != "" {
			logger.Infof("Auto-registering listener '%s' (Type: %s)", listenerName, statusMsg.Type)
			_, err = s.ListenerService.CreateListener(ctx, listenerName, statusMsg.Type, statusMsg.ConfigJson)
			if err != nil {
				logger.Errorf("Failed to auto-register listener: %v", err)
			}
		}
	}

	logger.Infof("Listener '%s' connected to control channel.", listenerName)

	// 2. 注册连接
	s.ListenerService.RegisterConnection(listenerName, stream)
	defer s.ListenerService.UnregisterConnection(listenerName)

	// 3. 循环接收状态更新
	for {
		statusMsg, err := stream.Recv()
		if err == io.EOF {
			logger.Infof("Listener '%s' disconnected (EOF).", listenerName)
			return nil
		}
		if err != nil {
			logger.Errorf("Error receiving status from listener '%s': %v", listenerName, err)
			return err
		}

		// 处理状态更新 (例如更新数据库状态)
		logger.Debugf("Listener '%s' status update: Active=%v, Beacons=%d, Error=%s", 
			listenerName, statusMsg.Active, statusMsg.ActiveBeacons, statusMsg.ErrorMessage)
            
        // TODO: Update database state based on received status
	}
}
