# Redis Streams Event Bus

> [中文文档](README.zh.md)

`github.com/nuln/agent-bus-redisstream` — Redis Streams Event Bus plugin for [nuln/agent-core](https://github.com/nuln/agent-core).

## Overview

| Field | Value |
|-------|-------|
| **Plugin Type** | `bus` |
| **Module** | `github.com/nuln/agent-bus-redisstream` |
| **Key Dependency** | `github.com/redis/go-redis/v9` |

## Installation

```bash
go get github.com/nuln/agent-bus-redisstream
```

Import the package in your `main.go` (side-effect import triggers `init()`):

```go
import _ "github.com/nuln/agent-bus-redisstream"
```

## Configuration

Configure via environment variables or the Web UI.  
See `RegisterPluginConfigSpec` in the plugin source for the full field list.

## Development

```bash
make fmt     # format code
make lint    # run golangci-lint
make test    # run tests
make build   # go build ./...
```

## License

MIT
