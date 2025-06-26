# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `danmu-core`, a Go-based system for monitoring and managing Douyin (TikTok) live streams. It provides a gRPC API for managing live stream monitoring tasks and handles real-time message processing from Douyin live streams.

## Key Architecture Components

- **gRPC Server**: Main API server (`internal/server/rpc_server.go`) exposing LiveService
- **Manager Layer**: Task management (`internal/manager/douyin_manager.go`) handles live stream monitoring lifecycle
- **Core Engine**: Live stream connection and message processing (`core/douyin/` directory)
- **Database Layer**: PostgreSQL with GORM for persistence (`internal/model/`)
- **Configuration**: INI-based config system (`conf/app.ini`, `setting/setting.go`)
- **Logging**: Structured logging with zerolog and file rotation

## Common Development Commands

### Build
```bash
# Build for Linux (default target)
make build-linux

# Build for Windows
make build-windows

# Install dependencies
make install
```

### Protocol Buffers
```bash
# Generate Go code from .proto files
make proto

# Manual protobuf generation
protoc --proto_path=protobuf --go_out=. protobuf/douyin.proto
```

### Database
- PostgreSQL database with schema `live`
- Connection configured in `conf/app.ini`
- Models use GORM ORM

### Running
```bash
# Run with default config
./douyinlive-linux

# Run with custom config
./douyinlive-linux -config /path/to/config.ini
```

## Development Notes

### Message Processing Flow
1. `DouyinManager` manages multiple `DouyinTask` instances
2. Each task wraps a `DouyinLive` connection to a specific live stream
3. Messages are processed through registered handlers in `internal/handler/`
4. Processed messages are stored via models in `internal/model/`

### gRPC Service
- Service definition: `protobuf/live_rpc.proto`
- Generated code: `generated/api/`
- Main operations: AddTask, DeleteTask, UpdateTask for live stream monitoring

### Configuration
- Uses INI format in `conf/app.ini`
- Sections: `[database]`, `[log]`, `[rpc]`
- Loaded via `github.com/go-ini/ini` package

### Concurrency Design
- Each live stream runs in its own goroutine
- Mutex-based synchronization for task management
- Periodic health checking with 15-minute intervals

### Dependencies
Key external packages:
- `google.golang.org/grpc` - gRPC server
- `gorm.io/gorm` - ORM
- `github.com/rs/zerolog` - Structured logging
- `github.com/gorilla/websocket` - WebSocket connections
- `github.com/dop251/goja` - JavaScript execution for message processing