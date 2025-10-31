# Listener & Beacon 标准化开发指南

## 1. 引言

本文档旨在为希望扩展 SimpleC2 框架的开发者提供一份清晰、标准的指南，用于开发新型的 `Listener` 服务及其配对的 `Beacon`。

本框架的核心设计思想是将核心业务逻辑 (`TeamServer`) 与通信传输层 (`Listener`) 解耦。`Listener` 是一个独立的、可执行的程序，它的唯一职责是作为一个“协议桥梁”：

- **面向 Beacon**: 实现一种特定的通信协议（如 DNS、SMB、ICMP 等），负责与 `Beacon` 进行直接通信。
- **面向 TeamServer**: 将 `Beacon` 的流量转换为标准的 gRPC 调用，与 `TeamServer` 进行通信。

遵循本指南，您可以轻松地添加新的隐蔽通信信道，而无需关心 `TeamServer` 的内部业务逻辑。

## 2. 核心概念

- **协议适配**: 您的 `Listener` 需要完整地处理与 `Beacon` 之间的协议细节。例如，如果是 DNS Listener，您需要监听 53 端口，解析 DNS 查询，并将数据从特定记录（如 TXT 记录）中提取出来。
- **gRPC 桥接**: 您的 `Listener` 在处理完原生协议流量后，必须使用项目提供的 `common` 包，通过 gRPC 与 `TeamServer` 进行通信。所有认证、加密和授权都由 `common` 包处理。
- **无状态**: `Listener` 自身应该是无状态的。所有关于 `Beacon` 和任务的状态都由 `TeamServer` 维护。

## 3. 开发步骤

以下是开发一个新 `Listener` 的完整步骤。我们将以开发一个 `dns Listener` 为例。

### 步骤 1: 创建目录结构

在项目根目录的 `listeners` 文件夹下，为您的新 `Listener` 创建一个以其协议命名的目录。

```bash
mkdir -p listeners/dns
```

### 步骤 2: 创建主程序和配置文件

1.  在 `listeners/dns/` 目录下，创建一个 `main.go` 文件。
2.  您的程序需要实现通过 `-config` 标志加载 YAML 配置文件的逻辑，并在文件不存在时生成一个默认版本。您可以参考 `listeners/http/main.go` 的实现。

### 步骤 3: 编写 `main` 函数

您的 `main.go` 文件将是 `Listener` 的入口。其主要结构应如下：

```go
package main

import (
	"log"

	"simplec2/listeners/common"
	"simplec2/pkg/config"
	// 您自己的协议处理库，例如 "github.com/miekg/dns"
)

var cfg config.ListenerConfig

func main() {
	// 1. 实现 -config 标志解析和默认配置生成

	// 2. 加载配置
	if err := config.LoadConfig("path/to/your/listener.yaml", &cfg); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 3. 使用 common 包连接到 TeamServer
	conn, err := common.ConnectToTeamServer(&cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer conn.Close()

	// 4. 设置并启动您的协议服务器
	// server := &dns.Server{Addr: cfg.Listener.Port, Net: "udp"}
	// server.Handler = &dnsHandler{}
	// log.Printf("DNS Listener starting on port %s", cfg.Listener.Port)
	// if err := server.ListenAndServe(); err != nil {
	// 	log.Fatalf("Failed to start DNS listener: %v", err)
	// }
}
```

### 步骤 4: 实现协议处理器

您需要编写一个处理器 (Handler) 来处理具体的协议逻辑。这个处理器将负责解析来自 `Beacon` 的原始请求，并调用 `common` 包提供的 gRPC 函数。

**注意**: 现有的 `http` listener 实现了一个完整的加密层。新的 listener **也必须实现类似的加密机制**，包括一个 `/handshake` 端点用于密钥交换，以及对所有后续流量的加解密。

## 4. Beacon 命令实现指南

`Beacon` 的核心职责是解析并执行从 `Listener` 收到的任务。以下是当前已定义的标准命令及其实现要求。

### CommandID 1: `shell`

- **描述**: 执行一个 shell 命令。
- **参数 (`Arguments`)**: 要执行的完整命令字符串 (e.g., `"whoami /all"`)。
- **输出 (`Output`)**: 命令执行后的 `stdout` 和 `stderr` 的组合输出。

### CommandID 2: `download`

- **描述**: 从 `TeamServer` 下载一个文件到 `Beacon` 所在的主机。
- **参数 (`Arguments`)**: 一个 JSON 字符串，包含 `dest_path` (目标路径) 和 `file_data` (Base64 编码的文件内容)。
- **输出 (`Output`)**: 一个表示操作结果的简单字符串，如 `"File downloaded successfully to [path]"` 或错误信息。

### CommandID 3: `upload`

- **描述**: 从 `Beacon` 所在的主机上传一个文件到 `TeamServer`。
- **参数 (`Arguments`)**: 要上传的文件的**绝对路径**字符串。
- **输出 (`Output`)**: **必须是所请求文件的原始二进制内容**。`TeamServer` 会将此内容保存到 `loot` 目录。

### CommandID 4: `exit`

- **描述**: 指示 `Beacon` 终止自身进程。
- **参数 (`Arguments`)**: 无。
- **输出 (`Output`)**: 无。

### CommandID 5: `sleep`

- **描述**: 调整 `Beacon` 的心跳间隔。
- **参数 (`Arguments`)**: 一个代表秒数的**字符串** (e.g., `"30"`)。
- **输出 (`Output`)**: 一个表示操作结果的简单字符串，如 `"Sleep interval updated to 30 seconds"`。

### CommandID 6: `browse`

- **描述**: 获取 `Beacon` 主机上指定目录的文件列表。
- **参数 (`Arguments`)**: 要浏览的目录的**绝对或相对路径**字符串。
- **输出 (`Output`)**: 一个由两部分组成的字符串，以换行符 (`\n`) 分隔：
    1.  **第一行**: `Beacon` **实际浏览的目录的绝对路径**。
    2.  **第二行及之后**: 一个包含多个 `FileInfo` 对象的 **JSON 字符串**。

- **`FileInfo` JSON 对象结构**:

  ```json
  {
    "name": "example.txt",
    "is_dir": false,
    "size": 1024,
    "last_mod_time": "2025-10-28T12:00:00Z" // RFC3339 格式
  }
  ```
