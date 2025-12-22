# Listener 开发指南

## 1. 简介

SimpleC2 的 Listener 是一个独立的进程，负责处理与 Beacon 的通信并将数据转发给 TeamServer。TeamServer 与 Listener 之间通过 gRPC (mTLS) 进行通信，实现了核心逻辑与通信协议的解耦。

目前的架构中，Listener 需要手动编译并配合配置文件运行。

## 2. Listener 的职责

1.  **协议转换**: 将 Beacon 的特定通信协议（如 HTTP, DNS）转换为 TeamServer 定义的标准 gRPC 接口调用。
2.  **加密/解密**: 处理 Beacon 与 Listener 之间的传输层加密（如 AES 在 HTTP Body 中）。
3.  **配置管理**: 读取本地配置文件 (`listener.yaml`) 以连接 TeamServer。

## 3. 开发步骤

以开发一个新的 `MyProtocol` Listener 为例：

### 3.1 目录结构
在 `listeners/` 目录下创建新目录 `listeners/myprotocol`。

### 3.2 配置文件
Listener 需要一个 `config.ListenerConfig` 结构来连接 TeamServer。通常使用 `pkg/config` 包加载 YAML。

### 3.3 核心代码 (`main.go`)

```go
package main

import (
    "simplec2/listeners/common"
    "simplec2/pkg/config"
    // ...
)

func main() {
    // 1. 加载配置
    var cfg config.ListenerConfig
    config.LoadConfig("listener.yaml", &cfg)

    // 2. 连接 TeamServer
    // common 包封装了 mTLS 连接逻辑
    conn, err := common.ConnectToTeamServer(&cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 3. 启动控制通道 (用于接收 TeamServer 的停止指令等)
    // 目前 ConfigJSON 主要用于简单的端口汇报，暂不支持复杂的动态配置
    common.StartControlChannel(&cfg, "MyProtocol", "{}", handleCommand)

    // 4. 启动你的协议服务
    startMyServer()
}

func handleCommand(cmd *bridge.ListenerCommand) {
    // 处理 START/STOP 等指令
}
```

### 3.4 调用 gRPC 接口

使用 `common.TSClient` 调用 TeamServer 接口：

*   **Beacon 上线**: `TSClient.StageBeacon(ctx, &bridge.StageBeaconRequest{...})`
*   **心跳/获取任务**: `TSClient.CheckInBeacon(ctx, &bridge.CheckInBeaconRequest{...})`
*   **回传结果**: `TSClient.PushBeaconOutput(ctx, &bridge.PushBeaconOutputRequest{...})`
*   **下载文件分片**: `TSClient.GetTaskedFileChunk(ctx, ...)`

## 4. 现有公共库 (`listeners/common`)

`simplec2/listeners/common` 提供了以下辅助功能：
*   `ConnectToTeamServer`: 建立 gRPC 连接。
*   `StartControlChannel`: 维持与 TS 的控制流。
*   `CreateAuthenticatedContext`: 创建带 API Key 的 gRPC Context。

## 5. 注意事项

*   **Tunnel 功能已移除**: 请忽略 `bridge.proto` 中关于 Tunnel 的定义。
*   **Jitter 支持**: Beacon 模型现已包含 `Jitter` 字段，Listener 在处理心跳时无需特殊处理，只需透传数据。