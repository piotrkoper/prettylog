# prettylog

A lightweight Go package that provides a human-readable, pretty-printed handler for the standard [`log/slog`](https://pkg.go.dev/log/slog) library (Go 1.21+).

Output format: `[HH:MM:SS.mmm]   LEVEL: message {"key":"value"}`

## Installation

```bash
go get github.com/piotrkoper/prettylog
```

## Quick Start

```go
package main

import (
    "log/slog"

    "github.com/piotrkoper/prettylog"
)

func main() {
    logger := slog.New(prettylog.NewHandler(nil))
    slog.SetDefault(logger)

    slog.Info("server started", "port", 8080)
    slog.Warn("high memory usage", "percent", 91)
    slog.Error("database connection failed", "host", "localhost")
}
```

Example output:

```
[10:05:23.001]    INFO: server started {"port":8080}
[10:05:23.002]    WARN: high memory usage {"percent":91}
[10:05:23.003]   ERROR: database connection failed {"host":"localhost"}
```

## API

### `NewHandler`

```go
func NewHandler(opts *slog.HandlerOptions) *Handler
```

Creates a handler that writes pretty-printed log lines to **stdout** with `WithOutputEmptyAttrs` enabled, so `{}` is appended even when a log record has no extra attributes. Pass `nil` for default options.

```go
handler := prettylog.NewHandler(&slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)
```

### `New`

```go
func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler
```

Creates a handler with full control over options. By default the handler has no writer set; use `WithDestinationWriter` to specify one.

```go
handler := prettylog.New(
    &slog.HandlerOptions{Level: slog.LevelInfo},
    prettylog.WithDestinationWriter(os.Stderr),
)
logger := slog.New(handler)
```

### Options

#### `WithDestinationWriter`

```go
func WithDestinationWriter(writer io.Writer) Option
```

Sets the output destination. Any `io.Writer` is accepted (e.g. `os.Stdout`, `os.Stderr`, a `*bytes.Buffer`).

```go
f, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
handler := prettylog.New(nil, prettylog.WithDestinationWriter(f))
```

#### `WithOutputEmptyAttrs`

```go
func WithOutputEmptyAttrs() Option
```

Appends `{}` to log lines that carry no extra attributes. Without this option, the JSON object is omitted entirely when a record has no attributes. Useful when downstream tooling expects a consistent trailing JSON object on every line.

```go
handler := prettylog.New(nil,
    prettylog.WithDestinationWriter(os.Stdout),
    prettylog.WithOutputEmptyAttrs(),
)
```

## Examples

### Setting a minimum log level

```go
handler := prettylog.NewHandler(&slog.HandlerOptions{
    Level: slog.LevelWarn,
})
logger := slog.New(handler)

logger.Debug("this will not appear")
logger.Info("this will not appear")
logger.Warn("this will appear")
logger.Error("this will appear")
```

### Using `WithAttrs` for persistent fields

```go
handler := prettylog.NewHandler(nil)
logger := slog.New(handler).With("service", "api", "version", "1.2.3")

logger.Info("request received", "method", "GET", "path", "/health")
// [10:05:23.001]    INFO: request received {"method":"GET","path":"/health","service":"api","version":"1.2.3"}
```

### Using groups

```go
handler := prettylog.NewHandler(nil)
logger := slog.New(handler).WithGroup("http")

logger.Info("request", "method", "POST", "status", 201)
// [10:05:23.001]    INFO: request {"http":{"method":"POST","status":201}}
```

### Customising attribute output with `ReplaceAttr`

```go
handler := prettylog.NewHandler(&slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // rename "msg" key to "message" in the JSON attrs
        if a.Key == slog.MessageKey {
            a.Key = "message"
        }
        return a
    },
})
logger := slog.New(handler)
```

### Writing to a file

```go
f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
if err != nil {
    log.Fatal(err)
}
defer f.Close()

handler := prettylog.New(nil, prettylog.WithDestinationWriter(f))
logger := slog.New(handler)
logger.Info("logging to file")
```

## Output Format

Each log line has the form:

```
[HH:MM:SS.mmm]   LEVEL: message {attributes}
```

| Field        | Description                                                         |
|--------------|---------------------------------------------------------------------|
| `timestamp`  | Current time formatted as `[HH:MM:SS.mmm]`                         |
| `level`      | Log level right-padded to 7 characters (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `message`    | The log message string                                              |
| `attributes` | Additional key/value pairs serialised as a JSON object (omitted when empty, unless `WithOutputEmptyAttrs` is used) |

## Credits

Inspired by [dusted-go/logging](https://github.com/dusted-go/logging/blob/main/prettylog/prettylog.go).

## License

See [LICENSE](LICENSE).