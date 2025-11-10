# SimpleC2 - 一个模块化的 C2 框架

**注意：这是一个非常前期版本的项目，并且代码均由大语言模型开发，目前可能存在非常多隐性的问题。**

SimpleC2 是一个轻量级、模块化、可扩展的C2框架，其核心设计在于将核心逻辑与通信层完全分离，使开发人员能够快速集成新的通信协议和 Beacon 类型。

## 核心设计思想

设计的整体思想来自于 [SimpleC2 框架设计文档](https://www.icewindy.cn/2025/11/09/SimpleC2%E6%A1%86%E6%9E%B6%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3/)。

- **分离式架构**: TeamServer 与 Listeners 解耦，允许独立部署和扩展。
- **gRPC 内部桥接**: Listener 通过一个安全高效的 gRPC 桥接与 TeamServer 进行通信，并采用 mTLS 进行双向认证。
- **易于扩展**: 无需修改 `TeamServer` 核心代码，即可轻松添加新的 `Listener` 类型 (如 DNS, SMB) 或 `Beacon` 命令。

## 构建与运行指南

本文档提供了编译和运行 SimpleC2 框架各个组件的说明。

### 环境要求

- Go (1.18+)
- Node.js (16+)
- `protoc` 编译器 (如果需要重新生成 gRPC 代码)

### 首次运行：生成所有必需的加密材料

在首次构建或运行任何组件之前，您必须生成所有用于 E2E 加密和 mTLS 通信的密钥与证书。项目提供了一个简化的命令来完成此操作：

```bash
make generate-keys
```

此命令将自动生成所有必需的文件，并将它们放置在 `certs/teamserver` 和 `certs/listener` 目录中，同时也会为 `certs/agent` 提供公钥。

### Makefile 工作流

项目包含一个 `Makefile` 以简化构建流程。

- `make generate-keys`: 生成所有必需的密钥和证书。
- `make` 或 `make all`: 构建所有组件 (`teamserver`, `listener_http`, 所有 `beacons`)。
- `make http`: 构建 HTTP listener 及其对应的 beacons。
- `make teamserver`: 仅构建 TeamServer。
- `make beacons-http`: 交叉编译 HTTP listener 的所有 beacon 版本。
- `make clean`: 从 `bin/` 目录中删除所有构建产物。

### 组件说明

#### 1. TeamServer

TeamServer 是 C2 的核心服务器。

- **配置**: 服务器通过 `teamserver.yaml` 文件进行配置。如果首次运行时找不到该文件，将自动生成一个默认配置。

  **数据库选项**:
  您可以在 `database` 部分选择使用 `sqlite` (默认) 或 `postgres`。

  - **SQLite (默认)**: 无需额外设置，数据库文件将根据配置中的 `path` 创建。
    ```yaml
    database:
      type: sqlite
      path: data/simplec2.db
    ```

  - **PostgreSQL**: 需要您提供一个有效的数据库连接字符串 (DSN)。
    ```yaml
    database:
      type: postgres
      dsn: "host=localhost user=postgres password=your_password dbname=simplec2 port=5432 sslmode=disable"
    ```

- **如何运行**:
  
  1.  **构建**: `make teamserver`
  2.  **复制证书**: 将 `certs/teamserver/s` 目录下所有文件**手动复制**到 `bin/teamserver/certs/` 目录下。
  3.  **运行**: 进入 `bin/teamserver/` 目录，然后执行：
  
  ```bash
  ./teamserver -config teamserver.yaml
  ```

#### 2. HTTP Listener

Listener 作为 Beacon 和 TeamServer 之间的桥梁。

- **配置**: Listener 通过 `listener.yaml` 文件进行配置。如果找不到该文件，将自动生成一个默认文件。
  
- **如何运行**:
  1.  **构建**: `make listener-http`
  2.  **复制证书**: 将 `certs/listener/` 目录**手动复制**到 `bin/listener_http/certs/` 目录下。
  3.  **运行**: 进入 `bin/listener_http/` 目录，然后执行：
  ```bash
  ./listener_http -config listener.yaml
  ```

#### 3. Http Beacon

Beacon 是运行在目标机器上的植入体。其 listener 的 URL 在编译时注入，RSA 公钥将使用同目录下 listener.pub 文件，请自行根据需要修改。

- **构建命令**:
  使用 `Makefile` 可以进行交叉编译。以下命令会将二进制文件放置在 `bin/beacons/` 目录中。

  ```bash
  make beacons-http LISTENER_URL=http://<your_c2_domain_or_ip>:8888
  ```

#### 4. Web UI

Web UI 是操作员的图形界面。

- **如何运行 (用于开发)**:

  ```bash
  cd webui
  npm run dev
  ```

  UI 将在 `http://localhost:5173` 上可用。

- **如何构建 (用于生产)**:

  ```bash
  cd webui
  npm run build
  ```

  静态文件将生成在 `webui/dist` 目录中。
