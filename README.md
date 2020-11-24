# Trace package for our systems

Trace packages for the different frameworks we use in our systems.

## Installation

1. Add dependency to go mod
2. Run go build/run/tidy

```bash
go get -u github.com/proemergotech/trace v0.3.0
```

## Error logging

The `trace.Error` function will log the error and error fields to the span 
if the error implements the `fielder` interface:

```go
    type fielder interface {
      Fields() []interface{}
    }
```

If the error wrapped other errors and implements the `causer` interface, 
the nested errors and their fields will be logged too.

```go
    type causer interface {
      Cause() error
    }
```

## Development

- install go
- check out project to: $GOPATH/src/github.com/proemergotech/trace
